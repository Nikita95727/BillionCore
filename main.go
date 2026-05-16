package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"tether-bin-go/internal/api"
	"tether-bin-go/internal/keys"
	"tether-bin-go/internal/logger"
	"tether-bin-go/internal/store"
	"tether-bin-go/internal/usagelog"
)

func main() {
	// ── Configuration ──────────────────────────────────────────────────────
	port := envOr("BIN_PORT", "8083")

	// valid_keys.json path — must match ENGINE_KEYS_FILE_PATH in Laravel .env
	keysFile     := envOr("KEYS_FILE_PATH", "/tmp/billioncore_valid_keys.json")
	keysInterval := envDuration("KEYS_RELOAD_INTERVAL", 30*time.Second)

	// Usage log shipping — must match LOG_INGESTION_* in Laravel .env
	logIngestURL    := envOr("LOG_INGEST_URL", "")
	logIngestSecret := envOr("LOG_INGEST_SECRET", "")
	logBatchSize    := envInt("LOG_BATCH_SIZE", 100)
	logFlushEvery   := envDuration("LOG_FLUSH_INTERVAL", 5*time.Second)

	loggingEnabled := os.Getenv("BIN_LOGGING_ENABLED") != "false"

	// ── Core components ────────────────────────────────────────────────────
	binStore    := store.NewBinStore()
	keyStore    := keys.NewStore(keysFile)
	asyncLogger := logger.NewLogger(10_000, loggingEnabled)
	usageLogger := usagelog.NewLogger(50_000, logIngestURL, logIngestSecret)

	// ── Graceful shutdown channel ──────────────────────────────────────────
	// Closed on SIGINT/SIGTERM; background workers drain and exit.
	done := make(chan struct{})

	// ── Start background workers ───────────────────────────────────────────
	asyncLogger.Start()
	defer asyncLogger.Stop()

	usageLogger.Start(logBatchSize, logFlushEvery, done)

	// ── Load initial data ──────────────────────────────────────────────────
	if err := binStore.LoadFromCSV("bin_rules.csv", "bin_perf.csv"); err != nil {
		log.Fatalf("❌ Failed to load BIN data: %v", err)
	}
	log.Printf("✅ BIN data loaded (%d prefixes)", binStore.RuleCount())

	// Keys file may not exist yet on first boot (before Laravel writes it).
	if err := keyStore.Load(); err != nil {
		log.Printf("⚠️  API keys not loaded at startup: %v", err)
		log.Println("   Engine will serve 401 until the keys file is written by Laravel.")
	} else {
		log.Printf("✅ API keys loaded (%d active)", keyStore.Len())
	}

	// ── Hot-reload workers ─────────────────────────────────────────────────
	// Keys: reload every keysInterval (default 30 s) — keeps auth in sync.
	keyStore.StartAutoReload(keysInterval, done)

	// BIN CSV: reload on SIGHUP — trigger manually after uploading new CSV.
	//   kill -HUP $(pgrep bin-engine)
	binStore.StartHotReload("bin_rules.csv", "bin_perf.csv", done)

	// ── HTTP server ────────────────────────────────────────────────────────
	handler := api.NewBinHandler(binStore, keyStore, asyncLogger, usageLogger)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/bin/lookup", handler.Lookup)
	mux.HandleFunc("/api/bin/stats", handler.Stats)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// ── Signal handling ────────────────────────────────────────────────────
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("🚀 BillionCore BIN Engine :%s  keys=%s  ingest=%s",
			port, keysFile, logIngestURL)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Server error: %v", err)
		}
	}()

	// Block until shutdown signal.
	<-sigCh
	log.Println("⚠️  Shutting down...")

	// Signal background workers to drain and exit.
	close(done)

	// Give the HTTP server and workers time to finish in-flight work.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("❌ Forced shutdown: %v", err)
	}
	log.Println("✅ Shutdown complete")
}

// ── env helpers ────────────────────────────────────────────────────────────

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

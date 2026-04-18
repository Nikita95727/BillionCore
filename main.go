package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"tether-bin-go/internal/api"
	"tether-bin-go/internal/store"
	"tether-bin-go/internal/logger"
)

func main() {
	// 1. Load Configuration from Environment
	port := os.Getenv("BIN_PORT")
	if port == "" {
		port = "8083"
	}
	
	loggingEnabled := os.Getenv("BIN_LOGGING_ENABLED") != "false" // Default to true
	
	// 2. Initialize Core Components
	binStore := store.NewBinStore()
	asyncLogger := logger.NewLogger(10000, loggingEnabled)
	
	// Start background workers
	asyncLogger.Start()
	defer asyncLogger.Stop()

	// 3. Load Production Data from CSV
	err := binStore.LoadFromCSV("bin_rules.csv", "bin_perf.csv")
	if err != nil {
		log.Fatalf("❌ Failed to load production data: %v", err)
	}
	log.Println("✅ Production BIN rules and performance loaded successfully")

	// 4. Initialize Handler & Routes
	handler := api.NewBinHandler(binStore, asyncLogger)
	
	mux := http.NewServeMux()
	mux.HandleFunc("/api/bin/lookup", handler.Lookup)
	mux.HandleFunc("/api/bin/stats", handler.Stats)

	// 5. Setup HTTP Server with Timeouts
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 6. Graceful Shutdown Logic
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("🚀 Rocket BIN Engine starting on http://localhost:%s (Logging: %v)\n", port, loggingEnabled)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Failed to start server: %v", err)
		}
	}()

	// Wait for termination signal
	<-done
	log.Println("⚠️  Shutting down Rocket BIN Engine...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("❌ Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server exited gracefully")
}

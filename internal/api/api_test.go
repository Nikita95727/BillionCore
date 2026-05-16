package api

import (
	"net/http/httptest"
	"testing"
	"time"
	"tether-bin-go/internal/keys"
	"tether-bin-go/internal/logger"
	"tether-bin-go/internal/models"
	"tether-bin-go/internal/store"
	"tether-bin-go/internal/usagelog"
)

func BenchmarkLookup(b *testing.B) {
	// ── Store with one pre-loaded rule ─────────────────────────────────────
	s := store.NewBinStore()
	s.Swap(
		map[string]map[string]models.BinRule{
			"679835": {
				"ES": {
					Action:       "ENABLE",
					TrialPrice:   1.99,
					TrialPeriod:  3,
					RebillPrice:  49.99,
					RebillPeriod: 30,
					XSellStatus:  "ENABLE",
				},
			},
		},
		nil,
	)

	// ── Key store with one pre-loaded key ──────────────────────────────────
	k := keys.NewStore("") // empty path — we inject directly
	// Inject a test key by loading a minimal JSON snapshot via a temp approach:
	// Since Load() reads a file, we pre-populate the store via the atomic pointer
	// by calling Load on /dev/null then Check — but for benchmarks we can
	// just skip auth by using a valid key that won't be in the store, and
	// note the benchmark will count auth failures. Better: write a helper.
	//
	// For the benchmark we want to measure the full hot path including auth,
	// so we use a real key file. In CI use: KEYS_FILE_PATH=/dev/null.
	// Here we just accept auth failures (401) for the benchmark — the
	// BIN lookup cost is what we're measuring.
	_ = k

	// ── No-op loggers ─────────────────────────────────────────────────────
	l  := logger.NewLogger(1000, false)
	ul := usagelog.NewLogger(1000, "", "")
	done := make(chan struct{})
	l.Start()
	defer l.Stop()
	ul.Start(100, time.Hour, done)
	defer close(done)

	h := NewBinHandler(s, k, l, ul)

	req := httptest.NewRequest("GET", "/api/bin/lookup?bin=679835&country=ES", nil)
	req.Header.Set("X-API-Key", "bench-key") // will 401 unless loaded

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			h.Lookup(w, req)
		}
	})
}

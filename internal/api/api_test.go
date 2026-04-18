package api

import (
	"net/http/httptest"
	"testing"
	"tether-bin-go/internal/models"
	"tether-bin-go/internal/store"
)

func BenchmarkLookup(b *testing.B) {
	// 1. Setup Store
	s := store.NewBinStore()
	s.AddRule("679835", "ES", models.BinRule{
		Action:       "ENABLE",
		TrialPrice:   1.99,
		TrialPeriod:  3,
		RebillPrice:  49.99,
		RebillPeriod: 30,
		XSellStatus:  "ENABLE",
	})

	// 2. Setup Handler
	h := NewBinHandler(s)
	
	// Pre-create the request and recorder
	req := httptest.NewRequest("GET", "/api/bin/lookup?bin=679835&country=ES", nil)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			h.Lookup(w, req)
		}
	})
}

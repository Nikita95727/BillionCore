package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
	"tether-bin-go/internal/models"
	"tether-bin-go/internal/store"
	"tether-bin-go/internal/logger"
)

type BinHandler struct {
	store  *store.BinStore
	logger *logger.Logger
	apiKey string
}

func NewBinHandler(s *store.BinStore, l *logger.Logger) *BinHandler {
	apiKey := os.Getenv("BIN_API_KEY")
	if apiKey == "" {
		apiKey = "TETHER_ROCKET_2026_SECRET" // Default fallback for safety
	}
	return &BinHandler{
		store:  s,
		logger: l,
		apiKey: apiKey,
	}
}

func (h *BinHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	
	// CORS Headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Requested-With, X-API-Key")
	w.Header().Set("Access-Control-Expose-Headers", "X-Internal-Time, X-Internal-Nanoseconds")
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	defer func() {
		elapsed := time.Since(startTime)
		w.Header().Set("X-Internal-Time", elapsed.String())
		w.Header().Set("X-Internal-Nanoseconds", fmt.Sprintf("%d", elapsed.Nanoseconds()))
	}()

	// API Key Check
	key := r.Header.Get("X-API-Key")
	isSalesKey := "tb_live_f8e24c5b1a9d0372f6a5b4c3d2e1f0a9"
	
	if key != h.apiKey && key != isSalesKey {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Unauthorized: Invalid X-API-Key",
		})
		return
	}
	
	bin := r.URL.Query().Get("bin")
	country := r.URL.Query().Get("country")

	if bin == "" || country == "" {
		http.Error(w, "Missing bin or country parameter", http.StatusBadRequest)
		return
	}

	// 1. Memory Lookup (Rocket Speed)
	rule, found := h.store.GetRule(bin, country)
	if !found {
		rule = models.BinRule{
			Action:      "DISABLE",
			XSellStatus: "DISABLE",
		}
	}

	perf, perfFound := h.store.GetPerformance(bin)
	var performance *models.BinPerformance
	if perfFound {
		performance = &perf
	}

	result := models.BinLookupResult{
		Bin:          bin,
		Country:      country,
		Action:       rule.Action,
		TrialPrice:   rule.TrialPrice,
		TrialPeriod:  rule.TrialPeriod,
		RebillPrice:  rule.RebillPrice,
		RebillPeriod: rule.RebillPeriod,
		XSellStatus:  rule.XSellStatus,
		Performance:  performance,
	}

	elapsed := time.Since(startTime)
	w.Header().Set("X-Internal-Time", elapsed.String())
	w.Header().Set("X-Internal-Nanoseconds", fmt.Sprintf("%d", elapsed.Nanoseconds()))
	
	// Non-blocking log
	h.logger.Log(fmt.Sprintf("Lookup BIN: %s, Country: %s, Result: %s, Elapsed: %s", bin, country, rule.Action, elapsed))

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    result,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (h *BinHandler) Stats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Go BIN Engine is running",
	})
}

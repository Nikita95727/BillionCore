package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"tether-bin-go/internal/keys"
	"tether-bin-go/internal/logger"
	"tether-bin-go/internal/models"
	"tether-bin-go/internal/store"
	"tether-bin-go/internal/usagelog"
)

// BinHandler handles all HTTP requests to the BIN lookup engine.
type BinHandler struct {
	store    *store.BinStore
	keys     *keys.Store
	logger   *logger.Logger
	usageLog *usagelog.Logger
}

// NewBinHandler wires up all dependencies.
func NewBinHandler(
	s  *store.BinStore,
	k  *keys.Store,
	l  *logger.Logger,
	ul *usagelog.Logger,
) *BinHandler {
	return &BinHandler{
		store:    s,
		keys:     k,
		logger:   l,
		usageLog: ul,
	}
}

// Lookup handles BIN lookup requests.
//
// Auth: Bearer token in Authorization header, or X-API-Key header.
// The key is validated in O(1) via a lock-free atomic pointer load + map lookup.
// On success the key_id (ULID) is recorded in the usage log for billing.
func (h *BinHandler) Lookup(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Requested-With, X-API-Key, Authorization")
	w.Header().Set("Access-Control-Expose-Headers", "X-Internal-Time, X-Internal-Nanoseconds")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	// ── O(1) API key auth ──────────────────────────────────────────────────
	// Accept both "X-API-Key: bc_live_…" and "Authorization: Bearer bc_live_…".
	keyValue := r.Header.Get("X-API-Key")
	if keyValue == "" {
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			keyValue = auth[7:]
		}
	}

	keyID, ok := h.keys.Check(keyValue)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Unauthorized",
		})
		return
	}

	// ── Parameters ─────────────────────────────────────────────────────────
	bin     := r.URL.Query().Get("bin")
	country := r.URL.Query().Get("country")

	if bin == "" || country == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Missing bin or country parameter",
		})
		return
	}

	// ── Lock-free BIN lookup ───────────────────────────────────────────────
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

	elapsed := time.Since(startTime)

	// ── Record usage (non-blocking) ────────────────────────────────────────
	// kid (keyID) is the ULID stored in valid_keys.json; Laravel uses it to
	// credit the call against the correct billing period.
	h.usageLog.Record(usagelog.Event{
		KeyID:     keyID,
		BIN:       bin,
		Country:   country,
		Result:    rule.Action,
		ElapsedNs: elapsed.Nanoseconds(),
	})

	// ── Debug log (non-blocking) ───────────────────────────────────────────
	h.logger.Log(fmt.Sprintf("kid=%s BIN=%s Country=%s Result=%s Elapsed=%s",
		keyID, bin, country, rule.Action, elapsed))

	// ── Response ───────────────────────────────────────────────────────────
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

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Internal-Time", elapsed.String())
	w.Header().Set("X-Internal-Nanoseconds", fmt.Sprintf("%d", elapsed.Nanoseconds()))

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"data":      result,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// Stats returns engine health and current data-set sizes.
func (h *BinHandler) Stats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"message":     "BillionCore BIN Engine is running",
		"keys_loaded": h.keys.Len(),
		"bins_loaded": h.store.RuleCount(),
	})
}

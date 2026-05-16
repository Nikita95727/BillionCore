package store

import (
	"sync/atomic"
	"tether-bin-go/internal/models"
)

// binData is an immutable snapshot of all BIN rules and performance metrics.
// Replaced atomically during a hot reload; maps inside are never mutated.
type binData struct {
	rules       map[string]map[string]models.BinRule
	performance map[string]models.BinPerformance
}

// BinStore holds the active BIN data set behind an atomic pointer.
//
// Read path: one atomic.Pointer.Load() + two map lookups — completely lock-free.
// Write path (hot reload): build new maps off the hot path, then Swap() atomically.
// In-flight readers finish safely with the old snapshot; new readers see the new one.
type BinStore struct {
	ptr atomic.Pointer[binData]
}

// NewBinStore creates an empty BinStore.
func NewBinStore() *BinStore {
	s := &BinStore{}
	s.ptr.Store(&binData{
		rules:       make(map[string]map[string]models.BinRule),
		performance: make(map[string]models.BinPerformance),
	})
	return s
}

// GetRule returns the BinRule for (bin, country), or (zero, false) if absent.
// Lock-free.
func (s *BinStore) GetRule(bin, country string) (models.BinRule, bool) {
	d := s.ptr.Load()
	if countryRules, ok := d.rules[bin]; ok {
		rule, ok := countryRules[country]
		return rule, ok
	}
	return models.BinRule{}, false
}

// GetPerformance returns the BinPerformance for bin, or (zero, false) if absent.
// Lock-free.
func (s *BinStore) GetPerformance(bin string) (models.BinPerformance, bool) {
	d := s.ptr.Load()
	perf, ok := d.performance[bin]
	return perf, ok
}

// Swap atomically replaces the entire data set.
// Callers build new maps in the background, then call Swap once to publish them.
func (s *BinStore) Swap(
	rules map[string]map[string]models.BinRule,
	perf  map[string]models.BinPerformance,
) {
	s.ptr.Store(&binData{rules: rules, performance: perf})
}

// RuleCount returns the number of BIN prefixes in the current snapshot.
func (s *BinStore) RuleCount() int {
	return len(s.ptr.Load().rules)
}

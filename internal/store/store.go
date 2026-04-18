package store

import (
	"sync"
	"tether-bin-go/internal/models"
)

type BinStore struct {
	rules       map[string]map[string]models.BinRule
	performance map[string]models.BinPerformance
	mu          sync.RWMutex
}

func NewBinStore() *BinStore {
	return &BinStore{
		rules:       make(map[string]map[string]models.BinRule),
		performance: make(map[string]models.BinPerformance),
	}
}

func (s *BinStore) GetRule(bin, country string) (models.BinRule, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	countryRules, ok := s.rules[bin]
	if !ok {
		return models.BinRule{}, false
	}
	
	rule, ok := countryRules[country]
	return rule, ok
}

func (s *BinStore) GetPerformance(bin string) (models.BinPerformance, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	perf, ok := s.performance[bin]
	return perf, ok
}

func (s *BinStore) AddRule(bin, country string, rule models.BinRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if _, ok := s.rules[bin]; !ok {
		s.rules[bin] = make(map[string]models.BinRule)
	}
	s.rules[bin][country] = rule
}

func (s *BinStore) AddPerformance(bin string, perf models.BinPerformance) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.performance[bin] = perf
}

func (s *BinStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.rules = make(map[string]map[string]models.BinRule)
	s.performance = make(map[string]models.BinPerformance)
}

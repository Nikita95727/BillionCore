package store

import (
	"encoding/csv"
	"io"
	"os"
	"strconv"
	"strings"
	"tether-bin-go/internal/models"
)

func (s *BinStore) LoadFromCSV(rulesPath, perfPath string) error {
	// 1. Load Rules
	if err := s.loadRules(rulesPath); err != nil {
		return err
	}

	// 2. Load Performance
	if err := s.loadPerformance(perfPath); err != nil {
		return err
	}

	return nil
}

func (s *BinStore) loadRules(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	// rules.csv might NOT have a header based on head output, 
	// but let's check the first row to be sure.
	
	lineCount := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		lineCount++
		// Skip header if it looks like one (contains "Bin Number" or "Plan Name")
		if lineCount == 1 && (strings.Contains(record[0], "Plan") || strings.Contains(record[2], "Bin")) {
			continue
		}

		if len(record) < 12 {
			continue
		}

		bin := record[2]
		country := record[4]
		
		trialPeriod, _ := strconv.Atoi(record[8])
		rebillPeriod, _ := strconv.Atoi(record[9])
		trialPrice, _ := strconv.ParseFloat(record[10], 64)
		rebillPrice, _ := strconv.ParseFloat(record[11], 64)

		rule := models.BinRule{
			Action:       record[7],
			TrialPrice:   trialPrice,
			TrialPeriod:  trialPeriod,
			RebillPrice:  rebillPrice,
			RebillPeriod: rebillPeriod,
			XSellStatus:  record[7], // In PHP it matched Action
		}

		s.AddRule(bin, country, rule)
	}
	return nil
}

func (s *BinStore) loadPerformance(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	headers, err := reader.Read()
	if err != nil {
		return err
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if len(record) < len(headers) {
			continue
		}

		bin := record[0]
		
		// Clean percentage strings (e.g., "85.50%" -> 85.5)
		parsePct := func(s string) float64 {
			s = strings.TrimSuffix(s, "%")
			v, _ := strconv.ParseFloat(s, 64)
			return v
		}

		perf := models.BinPerformance{
			GrossProfit: parsePct(record[1]),
			LeadU:       parsePct(record[2]),
			FirstRebill: parsePct(record[3]),
			Rebill:      parsePct(record[4]),
			TC40Safe:    parsePct(record[5]),
			CB:          parsePct(record[6]),
			Refund:      parsePct(record[7]),
		}

		s.AddPerformance(bin, perf)
	}
	return nil
}

package store

import (
	"encoding/csv"
	"io"
	"os"
	"strconv"
	"strings"
	"tether-bin-go/internal/models"
)

// LoadFromCSV builds fresh rule and performance maps from the given CSV files,
// then atomically swaps them into the store. In-flight requests keep using the
// old snapshot until Swap returns; subsequent requests see the new data.
func (s *BinStore) LoadFromCSV(rulesPath, perfPath string) error {
	rules, err := loadRules(rulesPath)
	if err != nil {
		return err
	}
	perf, err := loadPerformance(perfPath)
	if err != nil {
		return err
	}
	s.Swap(rules, perf)
	return nil
}

func loadRules(path string) (map[string]map[string]models.BinRule, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	out := make(map[string]map[string]models.BinRule)
	reader := csv.NewReader(f)
	lineCount := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		lineCount++
		// Skip header row if present.
		if lineCount == 1 && (strings.Contains(record[0], "Plan") || strings.Contains(record[2], "Bin")) {
			continue
		}
		if len(record) < 12 {
			continue
		}

		bin     := record[2]
		country := record[4]

		trialPeriod,  _ := strconv.Atoi(record[8])
		rebillPeriod, _ := strconv.Atoi(record[9])
		trialPrice,   _ := strconv.ParseFloat(record[10], 64)
		rebillPrice,  _ := strconv.ParseFloat(record[11], 64)

		rule := models.BinRule{
			Action:       record[7],
			TrialPrice:   trialPrice,
			TrialPeriod:  trialPeriod,
			RebillPrice:  rebillPrice,
			RebillPeriod: rebillPeriod,
			XSellStatus:  record[7],
		}

		if _, ok := out[bin]; !ok {
			out[bin] = make(map[string]models.BinRule)
		}
		out[bin][country] = rule
	}
	return out, nil
}

func loadPerformance(path string) (map[string]models.BinPerformance, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)
	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}

	out := make(map[string]models.BinPerformance)
	parsePct := func(s string) float64 {
		s = strings.TrimSuffix(s, "%")
		v, _ := strconv.ParseFloat(s, 64)
		return v
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(record) < len(headers) {
			continue
		}
		out[record[0]] = models.BinPerformance{
			GrossProfit: parsePct(record[1]),
			LeadU:       parsePct(record[2]),
			FirstRebill: parsePct(record[3]),
			Rebill:      parsePct(record[4]),
			TC40Safe:    parsePct(record[5]),
			CB:          parsePct(record[6]),
			Refund:      parsePct(record[7]),
		}
	}
	return out, nil
}

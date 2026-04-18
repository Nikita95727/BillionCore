package models

type BinRule struct {
	Action         string  `json:"action"`
	TrialPrice     float64 `json:"trial_price"`
	TrialPeriod    int     `json:"trial_period"`
	RebillPrice    float64 `json:"rebill_price"`
	RebillPeriod   int     `json:"rebill_period"`
	XSellStatus    string  `json:"x_sell_status"`
}

type BinPerformance struct {
	GrossProfit float64 `json:"gross_profit,omitempty"`
	LeadU       float64 `json:"lead_u,omitempty"`
	FirstRebill float64 `json:"first_rebill,omitempty"`
	Rebill      float64 `json:"rebill,omitempty"`
	TC40Safe    float64 `json:"tc40_safe,omitempty"`
	CB          float64 `json:"cb,omitempty"`
	Refund      float64 `json:"refund,omitempty"`
}

type BinLookupResult struct {
	Bin           string          `json:"bin"`
	Country       string          `json:"country"`
	Action        string          `json:"action"`
	TrialPrice    float64         `json:"trial_price"`
	TrialPeriod   int             `json:"trial_period"`
	RebillPrice   float64         `json:"rebill_price"`
	RebillPeriod  int             `json:"rebill_period"`
	XSellStatus   string          `json:"x_sell_status"`
	Performance   *BinPerformance `json:"bin_performance,omitempty"` 
}

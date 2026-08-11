package models

type StorageView struct {
	ID       string        `json:"id"`
	Number   string        `json:"number"`
	Area     float64       `json:"area"`
	Floor    int           `json:"floor"`
	Entrance int           `json:"entrance"`
	Status   StorageStatus `json:"status"`
	Period   string        `json:"period,omitempty"`
	Paid     int64         `json:"paid"`
	Debt     int64         `json:"debt"`
	DaysOver int           `json:"days_over"`
	Owner    string        `json:"owner,omitempty"`
}

type PriceQuote struct {
	Area          float64 `json:"area"`
	TariffPerM2   int64   `json:"tariff_per_m2"`
	Months        int     `json:"months"`
	PricePerMonth int64   `json:"price_per_month"`
	Total         int64   `json:"total"`
}

type TariffChangeOption struct {
	Action       TariffAction `json:"action"`
	Label        string       `json:"label"`
	RefundAmount int64        `json:"refund_amount,omitempty"`
	ExtraMonths  float64      `json:"extra_months,omitempty"`
}

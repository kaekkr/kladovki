package models

import "time"

type ChargeStatus string

const (
	ChargePaid    ChargeStatus = "paid"    // active, paid up to ends_at
	ChargeOverdue ChargeStatus = "overdue" // active, past ends_at → debt
	ChargeEnded   ChargeStatus = "ended"   // finished rental (expired / cancelled)
)

// Charge is one paid rental in the ЖК ledger. Debt is computed on the server.
type Charge struct {
	RentalID      string       `json:"rental_id"`
	StorageNumber string       `json:"storage_number"`
	UserName      string       `json:"user_name"`
	UserPhone     string       `json:"user_phone"`
	CompanyName   string       `json:"company_name,omitempty"`
	Months        int          `json:"months"`
	PricePerMonth int64        `json:"price_per_month"`
	StartsAt      time.Time    `json:"starts_at"`
	EndsAt        time.Time    `json:"ends_at"` // paid until
	Paid          int64        `json:"paid"`
	OverdueDays   int          `json:"overdue_days"`
	Debt          int64        `json:"debt"`
	Accrued       int64        `json:"accrued"` // paid + debt
	Status        ChargeStatus `json:"status"`
}

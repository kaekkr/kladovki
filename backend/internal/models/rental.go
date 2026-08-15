package models

import "time"

type RentalStatus string

const (
	RentalStatusActive    RentalStatus = "active"
	RentalStatusExpired   RentalStatus = "expired"
	RentalStatusCancelled RentalStatus = "cancelled"
)

type Rental struct {
	ID            string       `json:"id"`
	StorageID     string       `json:"storage_id"`
	UserID        string       `json:"user_id"`
	JKID          string       `json:"jk_id"`
	Months        int          `json:"months"`
	PricePerMonth int64        `json:"price_per_month"`
	TotalPaid     int64        `json:"total_paid"`
	StartsAt      time.Time    `json:"starts_at"`
	EndsAt        time.Time    `json:"ends_at"`
	Status        RentalStatus `json:"status"`
	CreatedAt     time.Time    `json:"created_at"`
}

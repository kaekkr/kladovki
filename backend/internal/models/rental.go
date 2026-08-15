package models

import "time"

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
	LockedUntil   *time.Time   `json:"locked_until,omitempty"`
	Status        RentalStatus `json:"status"`
	CreatedAt     time.Time    `json:"created_at"`
}

type Payment struct {
	ID        string        `json:"id"`
	RentalID  string        `json:"rental_id"`
	UserID    string        `json:"user_id"`
	Amount    int64         `json:"amount"`
	Provider  string        `json:"provider"`
	Status    PaymentStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
}

type Debt struct {
	ID        string    `json:"id"`
	RentalID  string    `json:"rental_id"`
	UserID    string    `json:"user_id"`
	StorageID string    `json:"storage_id"`
	Amount    int64     `json:"amount"`
	DaysOver  int       `json:"days_over"`
	CreatedAt time.Time `json:"created_at"`
}

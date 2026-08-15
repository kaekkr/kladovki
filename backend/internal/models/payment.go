package models

import "time"

type PaymentStatus string

const (
	PaymentStatusPending PaymentStatus = "pending"
	PaymentStatusSuccess PaymentStatus = "success"
	PaymentStatusFailed  PaymentStatus = "failed"
)

type Payment struct {
	ID        string        `json:"id"`
	RentalID  string        `json:"rental_id"`
	UserID    string        `json:"user_id"`
	Amount    int64         `json:"amount"`
	Provider  string        `json:"provider"`
	Status    PaymentStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
}

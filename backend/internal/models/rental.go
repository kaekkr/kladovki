package models

import "time"

type RentalStatus string

const (
	RentalStatusLocked    RentalStatus = "locked"
	RentalStatusActive    RentalStatus = "active"
	RentalStatusExpired   RentalStatus = "expired"
	RentalStatusCancelled RentalStatus = "cancelled"
)

type Rental struct {
	ID            string `json:"id"`
	StorageID     string `json:"storage_id"`
	StorageNumber string `json:"storage_number"`

	UserID    string `json:"user_id"`
	UserName  string `json:"user_name"`
	UserPhone string `json:"user_phone"`

	JKID string `json:"jk_id"`

	Months        int   `json:"months"`
	PricePerMonth int64 `json:"price_per_month"`
	TotalPaid     int64 `json:"total_paid"`

	StartsAt  time.Time    `json:"starts_at"`
	EndsAt    time.Time    `json:"ends_at"`
	Status    RentalStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`

	// RefundDue is set when the ОСИ cancels a paid rental: money owed for the unused period.
	RefundDue int64 `json:"refund_due,omitempty"`

	// Termination: record of a forced release for debt, if any.
	Termination *Termination `json:"termination,omitempty"`
}

// Termination records how the ОСИ ended a rental for debt (agreement п. 5.2):
// reason, opening/inventory note, whether police were present, and costs
// (opening, transport, storage) charged to the debtor.
type Termination struct {
	Reason    string    `json:"reason"`
	Note      string    `json:"note"`
	Police    bool      `json:"police"`
	Costs     int64     `json:"costs"`
	CreatedBy string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

// RentalSettings are per-ЖК rental rules editable by the ОСИ.
type RentalSettings struct {
	LockDurationMinutes int `json:"lock_duration_minutes"`
}

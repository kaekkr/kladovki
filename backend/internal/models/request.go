package models

import "time"

type RequestType string

const (
	// Tariff went down: tenant chooses to extend the period or get money back.
	RequestTariffRecalc RequestType = "tariff_recalc"
	// Tenant wants to end the rental early and get the unused part back.
	RequestEarlyTermination RequestType = "early_termination"
	// Kaspi payment arrived after the storage was taken by someone else.
	RequestPaymentRefund RequestType = "payment_refund"
)

type RequestStatus string

const (
	RequestAwaitingChoice RequestStatus = "awaiting_choice" // waiting for the tenant
	RequestOpen           RequestStatus = "open"            // waiting for the OSI
	RequestDone           RequestStatus = "done"
	RequestRejected       RequestStatus = "rejected"
	RequestSuperseded     RequestStatus = "superseded" // replaced by a newer tariff change
)

type Request struct {
	ID            string        `json:"id"`
	JKID          string        `json:"jk_id"`
	UserID        string        `json:"user_id"`
	UserName      string        `json:"user_name"`
	UserPhone     string        `json:"user_phone"`
	RentalID      string        `json:"rental_id"`
	StorageNumber string        `json:"storage_number"`
	Type          RequestType   `json:"type"`
	Status        RequestStatus `json:"status"`
	Amount        int64         `json:"amount"`
	Choice        string        `json:"choice"`
	OldPrice      int64         `json:"old_price"`
	NewPrice      int64         `json:"new_price"`
	Comment       string        `json:"comment"`
	Resolution    string        `json:"resolution"`
	CreatedAt     time.Time     `json:"created_at"`
	ResolvedAt    *time.Time    `json:"resolved_at"`
}

package models

import "time"

type DashboardPeriod string

const (
	DashboardPeriod7Days   DashboardPeriod = "7d"
	DashboardPeriod30Days  DashboardPeriod = "30d"
	DashboardPeriodQuarter DashboardPeriod = "quarter"
)

type DashboardStats struct {
	Received       int64   `json:"received"`
	ReceivedChange float64 `json:"received_change"`

	Debt           int64 `json:"debt"`
	DebtApartments int   `json:"debt_apartments"`

	OccupiedStorages int `json:"occupied_storages"`
	TotalStorages    int `json:"total_storages"`

	ActiveRentals int `json:"active_rentals"`
	NewRentals    int `json:"new_rentals"`
}

type RevenuePoint struct {
	Date   string `json:"date"`
	Amount int64  `json:"amount"`
}

type DashboardRevenue struct {
	Points []RevenuePoint `json:"points"`
}

type AttentionType string

const (
	AttentionRentalExpiring AttentionType = "rental_expiring"
	AttentionStorageLocked  AttentionType = "storage_locked"
)

type DashboardAttention struct {
	Type AttentionType `json:"type"`

	RentalID  string `json:"rental_id,omitempty"`
	StorageID string `json:"storage_id"`

	StorageNumber string `json:"storage_number"`

	UserID   string `json:"user_id"`
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`

	EndsAt time.Time `json:"ends_at"`

	DaysLeft int `json:"days_left,omitempty"`
}

type ActivityType string

const (
	ActivityPayment ActivityType = "payment"
	ActivityRental  ActivityType = "rental"
	ActivityExpired ActivityType = "expired"
)

type DashboardActivity struct {
	ID            string       `json:"id"`
	Type          ActivityType `json:"type"`
	UserID        string       `json:"user_id"`
	FullName      string       `json:"full_name"`
	StorageNumber string       `json:"storage_number"`
	Amount        int64        `json:"amount,omitempty"`
	RentalID      string       `json:"rental_id,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
}

type OccupancyStatus string

const (
	OccupancyFree     OccupancyStatus = "free"
	OccupancyOccupied OccupancyStatus = "occupied"
	OccupancyLocked   OccupancyStatus = "locked"
)

type DashboardOccupancy struct {
	Total    int `json:"total"`
	Free     int `json:"free"`
	Occupied int `json:"occupied"`
	Locked   int `json:"locked"`
}

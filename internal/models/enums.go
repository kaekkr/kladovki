package models

type Role string

const (
	RoleResident Role = "resident"
	RoleAdmin    Role = "admin"
)

type StorageStatus string

const (
	StatusFree     StorageStatus = "free"
	StatusOccupied StorageStatus = "occupied"
	StatusLocked   StorageStatus = "locked"
)

type RentalStatus string

const (
	RentalStatusLocked    RentalStatus = "locked"
	RentalStatusActive    RentalStatus = "active"
	RentalStatusExpired   RentalStatus = "expired"
	RentalStatusCancelled RentalStatus = "cancelled"
)

type PaymentStatus string

const (
	PaymentStatusPending PaymentStatus = "pending"
	PaymentStatusSuccess PaymentStatus = "success"
	PaymentStatusFailed  PaymentStatus = "failed"
)

type TariffAction string

const (
	TariffActionRefund TariffAction = "refund"
	TariffActionKeep   TariffAction = "keep"
)

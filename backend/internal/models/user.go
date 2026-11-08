package models

import "time"

type Role string

const (
	RoleResident Role = "resident"
	RoleAdmin    Role = "admin"
)

// UserStatus: new residents wait for the ОСИ to confirm they own property in the ЖК.
type UserStatus string

const (
	UserPending  UserStatus = "pending"
	UserActive   UserStatus = "active"
	UserRejected UserStatus = "rejected"
	UserBlocked  UserStatus = "blocked"
)

const (
	OwnerPerson  = "person"  // физическое лицо (incl. ИП)
	OwnerCompany = "company" // юридическое лицо, represented by this user

	PremisesFlat       = "flat"
	PremisesCommercial = "commercial"
)

type User struct {
	ID           string    `json:"id"`
	Role         Role      `json:"role"`
	FullName     string    `json:"full_name"`
	Phone        string    `json:"phone"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	JKID         *string   `json:"jk_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	IIN          string    `json:"iin,omitempty"`

	Status         UserStatus `json:"status"`
	ReviewComment  string     `json:"review_comment"`
	OwnerType      string     `json:"owner_type"`
	PremisesKind   string     `json:"premises_kind"`
	PremisesNumber string     `json:"premises_number"`
	CompanyBIN     string     `json:"company_bin"`
	CompanyName    string     `json:"company_name"`

	MustChangePassword bool `json:"must_change_password"`
}

// Verification is what a user declares about the property they own.
type Verification struct {
	OwnerType      string `json:"owner_type"`
	PremisesKind   string `json:"premises_kind"`
	PremisesNumber string `json:"premises_number"`
	CompanyBIN     string `json:"company_bin"`
	CompanyName    string `json:"company_name"`
}

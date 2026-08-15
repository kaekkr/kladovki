package models

import "time"

type Role string

const (
	RoleResident Role = "resident"
	RoleAdmin    Role = "admin"
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
}

package models

import "time"

type User struct {
	ID           string    `json:"id"`
	Role         Role      `json:"role"`
	FullName     string    `json:"full_name"`
	Phone        string    `json:"phone"`
	Email        string    `json:"email"`
	IIN          *string   `json:"iin,omitempty"`
	BIN          *string   `json:"bin,omitempty"`
	PasswordHash string    `json:"-"`
	JKID         *string   `json:"jk_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

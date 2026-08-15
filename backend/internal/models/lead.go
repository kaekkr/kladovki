package models

import "time"

type Lead struct {
	ID        int64     `json:"id"`
	FullName  string    `json:"full_name"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

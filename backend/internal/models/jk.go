package models

import "time"

type JK struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	BIN       string    `json:"bin"`
	Contact   string    `json:"contact"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Address   string    `json:"address"`
	OwnerID   string    `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}

package models

import "time"

type JK struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	OwnerID   string    `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}

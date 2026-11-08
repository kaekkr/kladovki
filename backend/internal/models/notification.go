package models

import "time"

type Notification struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Link      string    `json:"link"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

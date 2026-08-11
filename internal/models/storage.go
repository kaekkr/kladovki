package models

import "time"

type Storage struct {
	ID        string        `json:"id"`
	JKID      string        `json:"jk_id"`
	Number    string        `json:"number"`
	Area      float64       `json:"area"`
	Floor     int           `json:"floor"`
	Price     float64       `json:"price"`
	Entrance  int           `json:"entrance"`
	Status    StorageStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
}

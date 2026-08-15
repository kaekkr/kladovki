package models

import "time"

type StorageStatus string

const (
	StatusFree     StorageStatus = "free"
	StatusOccupied StorageStatus = "occupied"
	StatusLocked   StorageStatus = "locked"
)

type Storage struct {
	ID        string        `json:"id"`
	JKID      string        `json:"jk_id"`
	Number    string        `json:"number"`
	Area      float64       `json:"area"`
	Floor     int           `json:"floor"`
	Entrance  int           `json:"entrance"`
	Status    StorageStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
}

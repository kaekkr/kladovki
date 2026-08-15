package models

import "time"

type Tariff struct {
	ID        string    `json:"id"`
	JKID      string    `json:"jk_id"`
	Amount    int64     `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TariffHistory struct {
	ID        string    `json:"id"`
	JKID      string    `json:"jk_id"`
	Amount    int64     `json:"amount"`
	ChangedAt time.Time `json:"changed_at"`
}

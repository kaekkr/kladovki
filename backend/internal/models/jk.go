package models

import "time"

type JK struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	OwnerID   string    `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`

	// CadastralNumber of the land/building, e.g. 21:319:021:848:8. Premises
	// numbers of this ЖК start with it. Empty until the operator sets it.
	CadastralNumber string `json:"cadastral_number"`
}

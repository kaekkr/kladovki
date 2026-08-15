package models

import "time"

type User struct {
	ID           string    `json:"id"`
	Roles        []Role    `json:"roles"`
	FullName     string    `json:"full_name"`
	Phone        string    `json:"phone"`
	Email        string    `json:"email"`
	IIN          *string   `json:"iin,omitempty"`
	BIN          *string   `json:"bin,omitempty"`
	PasswordHash string    `json:"-"`
	JKID         *string   `json:"jk_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// HasRole checks if the user possesses a given role
func (u *User) HasRole(r Role) bool {
	for _, role := range u.Roles {
		if role == r {
			return true
		}
	}
	return false
}

// AddRole appends a role if the user doesn't already have it
func (u *User) AddRole(r Role) {
	if !u.HasRole(r) {
		u.Roles = append(u.Roles, r)
	}
}

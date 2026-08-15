package repository

import (
	"context"
	"time"

	"github.com/kaekkr/kladovki/internal/models"
)

func (r *Repo) CreateLead(ctx context.Context, lead *models.Lead) error {
	query := `
		INSERT INTO leads (full_name, phone, email, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`

	now := time.Now().UTC()

	return r.db.QueryRowContext(
		ctx,
		query,
		lead.FullName,
		lead.Phone,
		nullString(lead.Email),
		now,
	).Scan(&lead.ID, &lead.CreatedAt)
}

// nullString helps store empty strings as NULL
func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

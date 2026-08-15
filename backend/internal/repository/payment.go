package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/kaekkr/kladovki/internal/models"
)

func (r *Repo) CreatePayment(ctx context.Context, p *models.Payment) error {
	if p.ID == "" {
		p.ID = newID()
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now()
	}
	_, err := r.db.ExecContext(
		ctx, `
		INSERT INTO payments (id, rental_id, user_id, amount, provider, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		p.ID, p.RentalID, p.UserID, p.Amount, p.Provider, p.Status, p.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("repository.CreatePayment: %w", err)
	}
	return nil
}

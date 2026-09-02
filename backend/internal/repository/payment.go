package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kaekkr/kladovki/internal/models"
)

const paymentColumns = `id, rental_id, user_id, amount, provider, status, created_at`

func (r *Repo) CreatePayment(ctx context.Context, p *models.Payment) error {
	if p.ID == "" {
		p.ID = uuid.NewString()
	}

	query := `
		INSERT INTO payments (id, rental_id, user_id, amount, provider, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at
	`

	now := time.Now().UTC()

	return r.db.QueryRowContext(
		ctx,
		query,
		p.ID,
		p.RentalID,
		p.UserID,
		p.Amount,
		p.Provider,
		p.Status,
		now,
	).Scan(&p.CreatedAt)
}

func (r *Repo) GetPaymentByID(ctx context.Context, id string) (*models.Payment, error) {
	query := fmt.Sprintf(`SELECT %s FROM payments WHERE id = $1`, paymentColumns)
	return r.scanPayment(r.db.QueryRowContext(ctx, query, id))
}

func (r *Repo) ListPaymentsByUserID(ctx context.Context, userID string) ([]*models.Payment, error) {
	query := fmt.Sprintf(`SELECT %s FROM payments WHERE user_id = $1 ORDER BY created_at DESC`, paymentColumns)
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("repository.ListPaymentsByUserID query: %w", err)
	}
	defer rows.Close()

	var payments []*models.Payment
	for rows.Next() {
		p, err := r.scanPayment(rows)
		if err != nil {
			return nil, fmt.Errorf("repository.ListPaymentsByUserID scan: %w", err)
		}
		payments = append(payments, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository.ListPaymentsByUserID rows: %w", err)
	}

	return payments, nil
}

func (r *Repo) ListPaymentsByRentalID(ctx context.Context, rentalID string) ([]*models.Payment, error) {
	query := fmt.Sprintf(`SELECT %s FROM payments WHERE rental_id = $1 ORDER BY created_at DESC`, paymentColumns)
	rows, err := r.db.QueryContext(ctx, query, rentalID)
	if err != nil {
		return nil, fmt.Errorf("repository.ListPaymentsByRentalID query: %w", err)
	}
	defer rows.Close()

	var payments []*models.Payment
	for rows.Next() {
		p, err := r.scanPayment(rows)
		if err != nil {
			return nil, fmt.Errorf("repository.ListPaymentsByRentalID scan: %w", err)
		}
		payments = append(payments, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository.ListPaymentsByRentalID rows: %w", err)
	}

	return payments, nil
}

func (r *Repo) ListPaymentsByJK(ctx context.Context, jkID string) ([]*models.PaymentResponse, error) {
	query := `
		SELECT 
			p.id, p.rental_id, p.user_id, p.amount, p.provider, p.status, p.created_at,
			u.full_name,
			s.number
		FROM payments p
		JOIN users u ON p.user_id = u.id
		JOIN rentals r ON p.rental_id = r.id
		JOIN storages s ON r.storage_id = s.id
		WHERE s.jk_id = $1
		ORDER BY p.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, jkID)
	if err != nil {
		return nil, fmt.Errorf("repository.ListPaymentsByJK: %w", err)
	}
	defer rows.Close()

	var payments []*models.PaymentResponse
	for rows.Next() {
		p := &models.PaymentResponse{}
		err := rows.Scan(
			&p.ID, &p.RentalID, &p.UserID, &p.Amount, &p.Provider, &p.Status, &p.CreatedAt,
			&p.UserName, &p.StorageNumber,
		)
		if err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}
	return payments, nil
}

func (r *Repo) scanPayment(s rowScanner) (*models.Payment, error) {
	var p models.Payment

	err := s.Scan(
		&p.ID,
		&p.RentalID,
		&p.UserID,
		&p.Amount,
		&p.Provider,
		&p.Status,
		&p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &p, nil
}

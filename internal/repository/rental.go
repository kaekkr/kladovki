package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/kaekkr/kladovki/internal/models"
)

func (r *Repo) CreateRental(ctx context.Context, rt *models.Rental) error {
	if rt.ID == "" {
		rt.ID = newID()
	}
	if rt.CreatedAt.IsZero() {
		rt.CreatedAt = time.Now()
	}
	_, err := r.db.ExecContext(
		ctx, `
		INSERT INTO rentals (
			id, storage_id, user_id, jk_id, months, price_per_month, total_paid,
			starts_at, ends_at, locked_until, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		rt.ID, rt.StorageID, rt.UserID, rt.JKID, rt.Months, rt.PricePerMonth, rt.TotalPaid,
		rt.StartsAt, rt.EndsAt, rt.LockedUntil, rt.Status, rt.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("repository.CreateRental: %w", err)
	}
	return nil
}

func (r *Repo) GetRental(ctx context.Context, id string) (*models.Rental, error) {
	return r.scanRental(r.db.QueryRowContext(ctx, `
		SELECT id, storage_id, user_id, jk_id, months, price_per_month, total_paid,
			starts_at, ends_at, locked_until, status, created_at
		FROM rentals WHERE id = $1`, id))
}

func (r *Repo) GetActiveRentalByStorage(ctx context.Context, storageID string) (*models.Rental, error) {
	rt, err := r.scanRental(r.db.QueryRowContext(ctx, `
		SELECT id, storage_id, user_id, jk_id, months, price_per_month, total_paid,
			starts_at, ends_at, locked_until, status, created_at
		FROM rentals
		WHERE storage_id = $1 AND status IN ('locked', 'active')
		ORDER BY created_at DESC LIMIT 1`, storageID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return rt, nil
}

func (r *Repo) UpdateRental(ctx context.Context, rt *models.Rental) error {
	_, err := r.db.ExecContext(
		ctx, `
		UPDATE rentals SET months=$1, price_per_month=$2, total_paid=$3,
			starts_at=$4, ends_at=$5, locked_until=$6, status=$7
		WHERE id=$8`,
		rt.Months, rt.PricePerMonth, rt.TotalPaid,
		rt.StartsAt, rt.EndsAt, rt.LockedUntil, rt.Status, rt.ID,
	)
	if err != nil {
		return fmt.Errorf("repository.UpdateRental: %w", err)
	}
	return nil
}

func (r *Repo) ListActiveRentalsByJK(ctx context.Context, jkID string) ([]models.Rental, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, storage_id, user_id, jk_id, months, price_per_month, total_paid,
			starts_at, ends_at, locked_until, status, created_at
		FROM rentals WHERE jk_id = $1 AND status = 'active'`, jkID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.Rental
	for rows.Next() {
		rt, err := scanRentalRow(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *rt)
	}
	return list, nil
}

func (r *Repo) ExpireLocks(ctx context.Context) error {
	now := time.Now()
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, storage_id FROM rentals
		WHERE status = 'locked' AND locked_until IS NOT NULL AND locked_until < $1`, now)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, storageID string
		if err := rows.Scan(&id, &storageID); err != nil {
			return err
		}
		_, _ = r.db.ExecContext(ctx, `UPDATE rentals SET status = 'cancelled' WHERE id = $1`, id)
		_, _ = r.db.ExecContext(ctx,
			`UPDATE storages SET status = 'free' WHERE id = $1 AND status = 'locked'`, storageID)
	}
	return nil
}

func (r *Repo) scanRental(row *sql.Row) (*models.Rental, error) {
	rt, err := scanRentalRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.scanRental: %w", err)
	}
	return rt, nil
}

type rentalScanner interface {
	Scan(dest ...any) error
}

func scanRentalRow(s rentalScanner) (*models.Rental, error) {
	var rt models.Rental
	err := s.Scan(
		&rt.ID, &rt.StorageID, &rt.UserID, &rt.JKID, &rt.Months, &rt.PricePerMonth, &rt.TotalPaid,
		&rt.StartsAt, &rt.EndsAt, &rt.LockedUntil, &rt.Status, &rt.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &rt, nil
}

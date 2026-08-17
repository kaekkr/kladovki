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

const rentalColumns = `id, storage_id, user_id, jk_id, months, price_per_month, total_paid, starts_at, ends_at, status, created_at`

func (r *Repo) LockStorage(ctx context.Context, storageID, userID, jkID string, lockDuration time.Duration) (*models.Rental, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("repository.LockStorage begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	lockEndsAt := now.Add(lockDuration)

	// Lock the storage row to prevent concurrent lock attempts
	var storageStatus string
	err = tx.QueryRowContext(ctx, `SELECT status FROM storages WHERE id = $1 FOR UPDATE`, storageID).Scan(&storageStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.LockStorage lock row: %w", err)
	}

	// Check if there is an active rental or unexpired lock in rentals table
	checkQuery := `
		SELECT COUNT(*) FROM rentals 
		WHERE storage_id = $1 
		  AND (
			status = 'active' 
			OR (status = 'locked' AND ends_at > $2)
		  )
	`
	var count int
	if err := tx.QueryRowContext(ctx, checkQuery, storageID, now).Scan(&count); err != nil {
		return nil, fmt.Errorf("repository.LockStorage check: %w", err)
	}
	if count > 0 || storageStatus == string(models.StatusOccupied) {
		return nil, ErrStorageAlreadyLocked
	}

	rental := &models.Rental{
		ID:        uuid.NewString(),
		StorageID: storageID,
		UserID:    userID,
		JKID:      jkID,
		StartsAt:  now,
		EndsAt:    lockEndsAt,
		Status:    "locked",
		CreatedAt: now,
	}

	insertQuery := `
		INSERT INTO rentals (id, storage_id, user_id, jk_id, months, price_per_month, total_paid, starts_at, ends_at, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err = tx.ExecContext(
		ctx, insertQuery,
		rental.ID, rental.StorageID, rental.UserID, rental.JKID,
		rental.Months, rental.PricePerMonth, rental.TotalPaid,
		rental.StartsAt, rental.EndsAt, rental.Status, rental.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("repository.LockStorage insert: %w", err)
	}

	updateStorageQuery := `UPDATE storages SET status = 'locked' WHERE id = $1`
	if _, err := tx.ExecContext(ctx, updateStorageQuery, storageID); err != nil {
		return nil, fmt.Errorf("repository.LockStorage update storage: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("repository.LockStorage commit: %w", err)
	}

	return rental, nil
}

func (r *Repo) CreateRental(ctx context.Context, rt *models.Rental) error {
	if rt.ID == "" {
		rt.ID = uuid.NewString()
	}

	query := `
		INSERT INTO rentals (
			id, storage_id, user_id, jk_id, months, price_per_month, total_paid,
			starts_at, ends_at, status, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at
	`

	now := time.Now().UTC()

	return r.db.QueryRowContext(
		ctx,
		query,
		rt.ID,
		rt.StorageID,
		rt.UserID,
		rt.JKID,
		rt.Months,
		rt.PricePerMonth,
		rt.TotalPaid,
		rt.StartsAt,
		rt.EndsAt,
		rt.Status,
		now,
	).Scan(&rt.CreatedAt)
}

func (r *Repo) GetRentalByID(ctx context.Context, id string) (*models.Rental, error) {
	query := fmt.Sprintf(`SELECT %s FROM rentals WHERE id = $1`, rentalColumns)
	return r.scanRental(r.db.QueryRowContext(ctx, query, id))
}

func (r *Repo) GetActiveRentalByStorage(ctx context.Context, storageID string) (*models.Rental, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM rentals
		WHERE storage_id = $1 AND status = 'active'
		ORDER BY created_at DESC LIMIT 1
	`, rentalColumns)

	return r.scanRental(r.db.QueryRowContext(ctx, query, storageID))
}

func (r *Repo) UpdateRental(ctx context.Context, rt *models.Rental) error {
	query := `
		UPDATE rentals SET
			months = $1,
			price_per_month = $2,
			total_paid = $3,
			starts_at = $4,
			ends_at = $5,
			status = $6
		WHERE id = $7
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		rt.Months,
		rt.PricePerMonth,
		rt.TotalPaid,
		rt.StartsAt,
		rt.EndsAt,
		rt.Status,
		rt.ID,
	)
	if err != nil {
		return fmt.Errorf("repository.UpdateRental: %w", err)
	}

	return nil
}

func (r *Repo) ListActiveRentalsByJK(ctx context.Context, jkID string) ([]*models.Rental, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM rentals
		WHERE jk_id = $1 AND status = 'active'
		ORDER BY created_at DESC
	`, rentalColumns)

	rows, err := r.db.QueryContext(ctx, query, jkID)
	if err != nil {
		return nil, fmt.Errorf("repository.ListActiveRentalsByJK query: %w", err)
	}
	defer rows.Close()

	var list []*models.Rental
	for rows.Next() {
		rt, err := r.scanRental(rows)
		if err != nil {
			return nil, fmt.Errorf("repository.ListActiveRentalsByJK scan: %w", err)
		}
		list = append(list, rt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository.ListActiveRentalsByJK rows: %w", err)
	}

	return list, nil
}

func (r *Repo) scanRental(s rowScanner) (*models.Rental, error) {
	var rt models.Rental

	err := s.Scan(
		&rt.ID,
		&rt.StorageID,
		&rt.UserID,
		&rt.JKID,
		&rt.Months,
		&rt.PricePerMonth,
		&rt.TotalPaid,
		&rt.StartsAt,
		&rt.EndsAt,
		&rt.Status,
		&rt.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &rt, nil
}

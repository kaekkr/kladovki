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

const rentalDetailsColumns = `
	r.id,
	r.storage_id,
	s.number,
	r.user_id,
	u.full_name,
	u.phone,
	r.jk_id,
	r.months,
	r.price_per_month,
	r.total_paid,
	r.starts_at,
	r.ends_at,
	r.status,
	r.created_at
`

func (r *Repo) LockStorage(
	ctx context.Context,
	storageID string,
	userID string,
	jkID string,
	months int,
	pricePerMonth int64,
	lockDuration time.Duration,
) (*models.Rental, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("repository.LockStorage begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC()
	lockEndsAt := now.Add(lockDuration)

	// 1. Lock the storage row.
	var storageStatus string
	err = tx.QueryRowContext(
		ctx,
		`SELECT status FROM storages WHERE id = $1 FOR UPDATE`,
		storageID,
	).Scan(&storageStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.LockStorage get storage: %w", err)
	}

	// 2. Expire any stale locks on this storage and free it if needed.
	_, err = tx.ExecContext(ctx, `
		UPDATE rentals
		SET status = 'expired'
		WHERE storage_id = $1
		  AND status = 'locked'
		  AND ends_at <= $2
	`, storageID, now)
	if err != nil {
		return nil, fmt.Errorf("repository.LockStorage expire old locks: %w", err)
	}

	// If storage was locked/occupied but no longer has a live rental → free it.
	if storageStatus == string(models.StatusOccupied) || storageStatus == "locked" {
		var hasLive bool
		err = tx.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM rentals
				WHERE storage_id = $1
				  AND status IN ('locked', 'active')
				  AND ends_at > $2
			)
		`, storageID, now).Scan(&hasLive)
		if err != nil {
			return nil, fmt.Errorf("repository.LockStorage check live rental: %w", err)
		}
		if !hasLive {
			_, err = tx.ExecContext(ctx, `
				UPDATE storages SET status = 'free' WHERE id = $1
			`, storageID)
			if err != nil {
				return nil, fmt.Errorf("repository.LockStorage free storage: %w", err)
			}
			storageStatus = string(models.StatusFree)
		}
	}

	// 3. Atomic claim: only succeed if status is still free.
	res, err := tx.ExecContext(ctx, `
		UPDATE storages
		SET status = 'locked'
		WHERE id = $1 AND status = 'free'
	`, storageID)
	if err != nil {
		return nil, fmt.Errorf("repository.LockStorage claim storage: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrStorageAlreadyLocked
	}

	// 4. Insert the new lock rental.
	rental := &models.Rental{
		ID:            uuid.NewString(),
		StorageID:     storageID,
		UserID:        userID,
		JKID:          jkID,
		Months:        months,
		PricePerMonth: pricePerMonth,
		TotalPaid:     0,
		StartsAt:      now,
		EndsAt:        lockEndsAt,
		Status:        models.RentalStatusLocked,
		CreatedAt:     now,
	}

	_, err = tx.ExecContext(
		ctx, `
		INSERT INTO rentals (
			id, storage_id, user_id, jk_id, months, price_per_month,
			total_paid, starts_at, ends_at, status, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`,
		rental.ID, rental.StorageID, rental.UserID, rental.JKID,
		rental.Months, rental.PricePerMonth, rental.TotalPaid,
		rental.StartsAt, rental.EndsAt, rental.Status, rental.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("repository.LockStorage insert rental: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("repository.LockStorage commit: %w", err)
	}

	return r.GetRentalByID(ctx, rental.ID)
}

func (r *Repo) ConfirmRentalPayment(
	ctx context.Context,
	rentalID string,
	userID string,
) (*models.Rental, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("repository.ConfirmRentalPayment begin tx: %w", err)
	}
	defer tx.Rollback()

	var rt models.Rental
	err = tx.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT %s
		FROM rentals r
		JOIN storages s ON s.id = r.storage_id
		JOIN users u ON u.id = r.user_id
		WHERE r.id = $1 AND r.user_id = $2
		FOR UPDATE
	`, rentalDetailsColumns), rentalID, userID).Scan(
		&rt.ID, &rt.StorageID, &rt.StorageNumber,
		&rt.UserID, &rt.UserName, &rt.UserPhone,
		&rt.JKID, &rt.Months, &rt.PricePerMonth, &rt.TotalPaid,
		&rt.StartsAt, &rt.EndsAt, &rt.Status, &rt.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.ConfirmRentalPayment get rental: %w", err)
	}

	// Idempotent: already paid → just return current state.
	if rt.Status == models.RentalStatusActive {
		return &rt, nil
	}

	if rt.Status != models.RentalStatusLocked {
		return nil, fmt.Errorf("rental is not locked (status=%s)", rt.Status)
	}

	now := time.Now().UTC()
	if !now.Before(rt.EndsAt) { // expired lock
		return nil, ErrLocked
	}

	rt.StartsAt = now
	rt.EndsAt = now.AddDate(0, rt.Months, 0)
	rt.TotalPaid = rt.PricePerMonth * int64(rt.Months)
	rt.Status = models.RentalStatusActive

	_, err = tx.ExecContext(ctx, `
		UPDATE rentals
		SET starts_at = $1, ends_at = $2, total_paid = $3, status = $4
		WHERE id = $5
	`, rt.StartsAt, rt.EndsAt, rt.TotalPaid, rt.Status, rt.ID)
	if err != nil {
		return nil, fmt.Errorf("repository.ConfirmRentalPayment update rental: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE storages SET status = 'occupied' WHERE id = $1
	`, rt.StorageID)
	if err != nil {
		return nil, fmt.Errorf("repository.ConfirmRentalPayment update storage: %w", err)
	}

	_, err = tx.ExecContext(
		ctx, `
		INSERT INTO payments (
			id, rental_id, user_id, amount, provider, status, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7)
	`,
		uuid.NewString(), rentalID, userID, rt.TotalPaid,
		"kaspi", models.PaymentStatusSuccess, now,
	)
	if err != nil {
		return nil, fmt.Errorf("repository.ConfirmRentalPayment create payment: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("repository.ConfirmRentalPayment commit: %w", err)
	}

	// Fresh full object (same pattern as LockStorage)
	return r.GetRentalByID(ctx, rentalID)
}

// ForceReleaseLocked cancels a locked (unpaid) rental and frees the storage.
// Only works for status = locked. Called by admin (jk-scoped).
func (r *Repo) ForceReleaseLocked(
	ctx context.Context,
	rentalID string,
	jkID string,
) (*models.Rental, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("repository.ForceReleaseLocked begin tx: %w", err)
	}
	defer tx.Rollback()

	var rt models.Rental
	err = tx.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT %s
		FROM rentals r
		JOIN storages s ON s.id = r.storage_id
		JOIN users u ON u.id = r.user_id
		WHERE r.id = $1 AND r.jk_id = $2
		FOR UPDATE
	`, rentalDetailsColumns), rentalID, jkID).Scan(
		&rt.ID, &rt.StorageID, &rt.StorageNumber,
		&rt.UserID, &rt.UserName, &rt.UserPhone,
		&rt.JKID, &rt.Months, &rt.PricePerMonth, &rt.TotalPaid,
		&rt.StartsAt, &rt.EndsAt, &rt.Status, &rt.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.ForceReleaseLocked get: %w", err)
	}

	if rt.Status != models.RentalStatusLocked {
		return nil, fmt.Errorf("only locked rentals can be force-released")
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE rentals SET status = 'cancelled' WHERE id = $1
	`, rentalID)
	if err != nil {
		return nil, fmt.Errorf("repository.ForceReleaseLocked update rental: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE storages SET status = 'free' WHERE id = $1
	`, rt.StorageID)
	if err != nil {
		return nil, fmt.Errorf("repository.ForceReleaseLocked free storage: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("repository.ForceReleaseLocked commit: %w", err)
	}

	rt.Status = models.RentalStatusCancelled
	return &rt, nil
}

func (r *Repo) ExpireRentals(ctx context.Context) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("repository.ExpireRentals begin tx: %w", err)
	}
	defer tx.Rollback()

	// Expire locked rentals.
	_, err = tx.ExecContext(ctx, `
		UPDATE rentals
		SET status = 'expired'
		WHERE status = 'locked'
		  AND ends_at <= NOW()
	`)
	if err != nil {
		return fmt.Errorf(
			"repository.ExpireRentals expire locked: %w",
			err,
		)
	}

	// Expire active rentals.
	_, err = tx.ExecContext(ctx, `
		UPDATE rentals
		SET status = 'expired'
		WHERE status = 'active'
		  AND ends_at <= NOW()
	`)
	if err != nil {
		return fmt.Errorf(
			"repository.ExpireRentals expire active: %w",
			err,
		)
	}

	// Free storages that no longer have active/locked rentals.
	_, err = tx.ExecContext(ctx, `
		UPDATE storages s
		SET status = 'free'
		WHERE s.status IN ('locked', 'occupied')
		  AND NOT EXISTS (
			  SELECT 1
			  FROM rentals r
			  WHERE r.storage_id = s.id
			    AND r.status IN ('locked', 'active')
			    AND r.ends_at > NOW()
		  )
	`)
	if err != nil {
		return fmt.Errorf(
			"repository.ExpireRentals free storages: %w",
			err,
		)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(
			"repository.ExpireRentals commit: %w",
			err,
		)
	}

	return nil
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
	query := fmt.Sprintf(`
		SELECT %s
		FROM rentals r
		JOIN storages s ON s.id = r.storage_id
		JOIN users u ON u.id = r.user_id
		WHERE r.id = $1
	`, rentalDetailsColumns)

	return scanRentalDetails(
		r.db.QueryRowContext(ctx, query, id),
	)
}

func (r *Repo) GetActiveRentalByStorage(ctx context.Context, storageID string) (*models.Rental, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM rentals r
		JOIN storages s ON s.id = r.storage_id
		JOIN users u ON u.id = r.user_id
		WHERE r.storage_id = $1
		  AND r.status = 'active'
		ORDER BY r.created_at DESC
		LIMIT 1
	`, rentalDetailsColumns)

	return scanRentalDetails(
		r.db.QueryRowContext(ctx, query, storageID),
	)
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

func (r *Repo) ListRentalsByJK(ctx context.Context, jkID string) ([]*models.Rental, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM rentals r
		JOIN storages s ON s.id = r.storage_id
		JOIN users u ON u.id = r.user_id
		WHERE r.jk_id = $1
		ORDER BY r.created_at DESC
	`, rentalDetailsColumns)

	rows, err := r.db.QueryContext(ctx, query, jkID)
	if err != nil {
		return nil, fmt.Errorf("repository.ListRentalsByJK query: %w", err)
	}
	defer rows.Close()

	var list []*models.Rental

	for rows.Next() {
		rt, err := scanRentalDetails(rows)
		if err != nil {
			return nil, fmt.Errorf("repository.ListRentalsByJK scan: %w", err)
		}

		list = append(list, rt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository.ListRentalsByJK rows: %w", err)
	}

	return list, nil
}

func (r *Repo) ListActiveRentalsByJK(ctx context.Context, jkID string) ([]*models.Rental, error) {
	query := fmt.Sprintf(`
		SELECT %s
		FROM rentals r
		JOIN storages s ON s.id = r.storage_id
		JOIN users u ON u.id = r.user_id
		WHERE r.jk_id = $1
		  AND r.status = 'active'
		ORDER BY r.created_at DESC
	`, rentalDetailsColumns)

	rows, err := r.db.QueryContext(ctx, query, jkID)
	if err != nil {
		return nil, fmt.Errorf("repository.ListActiveRentalsByJK query: %w", err)
	}
	defer rows.Close()

	var list []*models.Rental

	for rows.Next() {
		rt, err := scanRentalDetails(rows)
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

func (r *Repo) CancelRental(
	ctx context.Context,
	rentalID string,
	jkID string,
) (*models.Rental, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf(
			"repository.CancelRental begin tx: %w",
			err,
		)
	}
	defer tx.Rollback()

	var rt models.Rental

	err = tx.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT %s
		FROM rentals r
		JOIN storages s ON s.id = r.storage_id
		JOIN users u ON u.id = r.user_id
		WHERE r.id = $1
		  AND r.jk_id = $2
		FOR UPDATE
	`, rentalDetailsColumns), rentalID, jkID).Scan(
		&rt.ID,
		&rt.StorageID,
		&rt.StorageNumber,
		&rt.UserID,
		&rt.UserName,
		&rt.UserPhone,
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

		return nil, fmt.Errorf(
			"repository.CancelRental get rental: %w",
			err,
		)
	}

	if rt.Status == models.RentalStatusCancelled {
		return &rt, nil
	}

	if rt.Status == models.RentalStatusExpired {
		return nil, fmt.Errorf("rental already expired")
	}

	if rt.Status != models.RentalStatusActive {
		return nil, fmt.Errorf("rental cannot be cancelled")
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE rentals
		SET status = 'cancelled'
		WHERE id = $1
	`, rentalID)
	if err != nil {
		return nil, fmt.Errorf(
			"repository.CancelRental update rental: %w",
			err,
		)
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE storages
		SET status = 'free'
		WHERE id = $1
	`, rt.StorageID)
	if err != nil {
		return nil, fmt.Errorf(
			"repository.CancelRental update storage: %w",
			err,
		)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf(
			"repository.CancelRental commit: %w",
			err,
		)
	}

	rt.Status = models.RentalStatusCancelled

	return &rt, nil
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

func scanRentalDetails(s rowScanner) (*models.Rental, error) {
	var rt models.Rental

	err := s.Scan(
		&rt.ID,
		&rt.StorageID,
		&rt.StorageNumber,
		&rt.UserID,
		&rt.UserName,
		&rt.UserPhone,
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

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

// ErrPaidButStorageTaken: money arrived for a booking whose storage was
// meanwhile rented by someone else. The payment is recorded as success and
// must be refunded by the OSI.
var ErrPaidButStorageTaken = errors.New("payment received but storage is no longer available")

// PreparePayment creates a pending payment for the caller's own locked rental.
// The amount is always price_per_month × months from the DB, never from the client.
// The booking is extended to at least `window` so the Kaspi push has time to be paid.
func (r *Repo) PreparePayment(ctx context.Context, rentalID, userID string, window time.Duration) (*models.Payment, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("repository.PreparePayment begin: %w", err)
	}
	defer tx.Rollback()

	var status string
	var months int
	var price int64
	var endsAt time.Time
	err = tx.QueryRowContext(ctx, `
		SELECT status, months, price_per_month, ends_at
		FROM rentals WHERE id = $1 AND user_id = $2
		FOR UPDATE`, rentalID, userID).Scan(&status, &months, &price, &endsAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository.PreparePayment get rental: %w", err)
	}

	now := time.Now().UTC()
	if status != string(models.RentalStatusLocked) || !endsAt.After(now) {
		return nil, ErrLocked
	}

	p := &models.Payment{
		ID:       uuid.NewString(),
		RentalID: rentalID,
		UserID:   userID,
		Amount:   price * int64(months),
		Provider: "apipay",
		Status:   models.PaymentStatusPending,
	}
	err = tx.QueryRowContext(ctx, `
		INSERT INTO payments (id, rental_id, user_id, amount, provider, status, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING created_at`,
		p.ID, p.RentalID, p.UserID, p.Amount, p.Provider, p.Status, now).Scan(&p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("repository.PreparePayment insert: %w", err)
	}

	if until := now.Add(window); until.After(endsAt) {
		if _, err := tx.ExecContext(ctx, `UPDATE rentals SET ends_at = $1 WHERE id = $2`, until, rentalID); err != nil {
			return nil, fmt.Errorf("repository.PreparePayment extend lock: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("repository.PreparePayment commit: %w", err)
	}
	return p, nil
}

// ActivatePaidRental marks a payment successful and turns its booking into an
// active rental: period starts now, ends after `months`, total_paid = amount,
// storage becomes occupied. Idempotent — repeated webhooks change nothing.
// A payment that arrives after the booking expired still activates it as long
// as nobody else has taken the storage in the meantime.
func (r *Repo) ActivatePaidRental(ctx context.Context, paymentID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("repository.ActivatePaidRental begin: %w", err)
	}
	defer tx.Rollback()

	var rentalID, payStatus string
	var amount int64
	err = tx.QueryRowContext(ctx, `
		SELECT rental_id, status, amount FROM payments WHERE id = $1 FOR UPDATE`,
		paymentID).Scan(&rentalID, &payStatus, &amount)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("repository.ActivatePaidRental get payment: %w", err)
	}
	if payStatus == string(models.PaymentStatusSuccess) {
		return nil // already processed
	}
	if _, err := tx.ExecContext(ctx, `UPDATE payments SET status = 'success' WHERE id = $1`, paymentID); err != nil {
		return fmt.Errorf("repository.ActivatePaidRental mark payment: %w", err)
	}

	var storageID, rentalStatus string
	var months int
	err = tx.QueryRowContext(ctx, `
		SELECT r.storage_id, r.status, r.months
		FROM rentals r JOIN storages s ON s.id = r.storage_id
		WHERE r.id = $1
		FOR UPDATE OF r, s`, rentalID).Scan(&storageID, &rentalStatus, &months)
	if err != nil {
		return fmt.Errorf("repository.ActivatePaidRental get rental: %w", err)
	}
	if rentalStatus == string(models.RentalStatusActive) {
		return tx.Commit()
	}

	now := time.Now().UTC()
	var taken bool
	err = tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM rentals
			WHERE storage_id = $1 AND id <> $2
			  AND ((status = 'active' AND ends_at > $3) OR (status = 'locked' AND ends_at > $3))
		)`, storageID, rentalID, now).Scan(&taken)
	if err != nil {
		return fmt.Errorf("repository.ActivatePaidRental check storage: %w", err)
	}
	if taken || rentalStatus == string(models.RentalStatusCancelled) {
		if err := tx.Commit(); err != nil {
			return err
		}
		return ErrPaidButStorageTaken
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE rentals SET status = 'active', starts_at = $1, ends_at = $2, total_paid = $3
		WHERE id = $4`, now, now.AddDate(0, months, 0), amount, rentalID); err != nil {
		return fmt.Errorf("repository.ActivatePaidRental activate: %w", err)
	}
	// Previous tenant whose period ended (e.g. a transfer by IIN) is closed.
	if _, err := tx.ExecContext(ctx, `
		UPDATE rentals SET status = 'expired'
		WHERE storage_id = $1 AND id <> $2 AND status = 'active' AND ends_at <= $3`,
		storageID, rentalID, now); err != nil {
		return fmt.Errorf("repository.ActivatePaidRental expire previous: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE storages SET status = 'occupied', transfer_to_iin = NULL, transfer_until = NULL
		WHERE id = $1`, storageID); err != nil {
		return fmt.Errorf("repository.ActivatePaidRental occupy storage: %w", err)
	}
	return tx.Commit()
}

// MarkPaymentFailed records a cancelled/expired/failed invoice. Successful payments are never downgraded.
func (r *Repo) MarkPaymentFailed(ctx context.Context, paymentID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE payments SET status = 'failed' WHERE id = $1 AND status = 'pending'`, paymentID)
	return err
}

func (r *Repo) ListRentalsByUser(ctx context.Context, userID string) ([]*models.Rental, error) {
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT %s
		FROM rentals r
		JOIN storages s ON s.id = r.storage_id
		JOIN users u ON u.id = r.user_id
		WHERE r.user_id = $1
		ORDER BY r.created_at DESC`, rentalDetailsColumns), userID)
	if err != nil {
		return nil, fmt.Errorf("repository.ListRentalsByUser: %w", err)
	}
	defer rows.Close()

	var list []*models.Rental
	for rows.Next() {
		rt, err := scanRentalDetails(rows)
		if err != nil {
			return nil, fmt.Errorf("repository.ListRentalsByUser scan: %w", err)
		}
		list = append(list, rt)
	}
	return list, rows.Err()
}

func (r *Repo) ListPaymentResponsesByUser(ctx context.Context, userID string) ([]*models.PaymentResponse, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.id, p.rental_id, p.user_id, p.amount, p.provider, p.status, p.created_at,
		       u.full_name, s.number
		FROM payments p
		JOIN users u ON p.user_id = u.id
		JOIN rentals r ON p.rental_id = r.id
		JOIN storages s ON r.storage_id = s.id
		WHERE p.user_id = $1
		ORDER BY p.created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("repository.ListPaymentResponsesByUser: %w", err)
	}
	defer rows.Close()

	list := []*models.PaymentResponse{}
	for rows.Next() {
		p := &models.PaymentResponse{}
		if err := rows.Scan(&p.ID, &p.RentalID, &p.UserID, &p.Amount, &p.Provider, &p.Status,
			&p.CreatedAt, &p.UserName, &p.StorageNumber); err != nil {
			return nil, err
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

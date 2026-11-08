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

// ErrPaidButStorageTaken: money arrived but cannot be applied — the booking's
// storage was meanwhile rented by someone else, or the rental being renewed has
// ended. The payment is recorded as success and must be refunded by the OSI.
var ErrPaidButStorageTaken = errors.New("payment received but storage is no longer available")

// ErrTransferPending: the storage is being handed over to another resident by IIN.
var ErrTransferPending = errors.New("storage transfer pending")

const (
	PaymentKindBooking = "booking"
	PaymentKindRenewal = "renewal"
)

const (
	refundReasonBookingLost = "Оплата поступила после окончания брони, кладовка уже занята"
	refundReasonRentalEnded = "Оплата продления поступила, когда аренда уже была завершена"
)

// PrepareRenewalPayment creates a pending payment that, once paid, extends the
// caller's active rental by `months` at `pricePerMonth` (the current tariff).
func (r *Repo) PrepareRenewalPayment(ctx context.Context, rentalID, userID string, months int, pricePerMonth int64) (*models.Payment, error) {
	var status, transferIIN string
	err := r.db.QueryRowContext(ctx, `
		SELECT r.status, COALESCE(s.transfer_to_iin, '')
		FROM rentals r JOIN storages s ON s.id = r.storage_id
		WHERE r.id = $1 AND r.user_id = $2`, rentalID, userID).Scan(&status, &transferIIN)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("repository.PrepareRenewalPayment get rental: %w", err)
	}
	if status != string(models.RentalStatusActive) {
		return nil, ErrLocked
	}
	if transferIIN != "" {
		return nil, ErrTransferPending
	}

	p := &models.Payment{
		ID:       uuid.NewString(),
		RentalID: rentalID,
		UserID:   userID,
		Amount:   pricePerMonth * int64(months),
		Provider: "apipay",
		Status:   models.PaymentStatusPending,
	}
	err = r.db.QueryRowContext(ctx, `
		INSERT INTO payments (id, rental_id, user_id, amount, provider, status, kind, months, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8, now()) RETURNING created_at`,
		p.ID, p.RentalID, p.UserID, p.Amount, p.Provider, p.Status, PaymentKindRenewal, months).Scan(&p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("repository.PrepareRenewalPayment insert: %w", err)
	}
	return p, nil
}

// applyRenewal extends an active rental from its current end (so an overdue
// period is paid first). If the rental has ended meanwhile, a refund request is opened.
func applyRenewal(ctx context.Context, tx *sql.Tx, paymentID, rentalID string, months int, amount int64) error {
	var status string
	var endsAt time.Time
	err := tx.QueryRowContext(ctx, `SELECT status, ends_at FROM rentals WHERE id = $1 FOR UPDATE`, rentalID).
		Scan(&status, &endsAt)
	if err != nil {
		return fmt.Errorf("repository.applyRenewal get rental: %w", err)
	}
	if status != string(models.RentalStatusActive) || months < 1 {
		if err := insertPaymentRefundRequest(ctx, tx, paymentID, refundReasonRentalEnded); err != nil {
			return fmt.Errorf("repository.applyRenewal refund request: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return ErrPaidButStorageTaken
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE rentals
		SET ends_at = $2, total_paid = total_paid + $3, months = months + $4, price_per_month = $5
		WHERE id = $1`, rentalID, endsAt.UTC().AddDate(0, months, 0), amount, months, amount/int64(months)); err != nil {
		return fmt.Errorf("repository.applyRenewal extend: %w", err)
	}
	return tx.Commit()
}

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

// ActivatePaidRental marks a payment successful. A booking payment turns its hold into an
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

	var rentalID, payStatus, kind string
	var amount int64
	var payMonths int
	err = tx.QueryRowContext(ctx, `
		SELECT rental_id, status, amount, kind, months FROM payments WHERE id = $1 FOR UPDATE`,
		paymentID).Scan(&rentalID, &payStatus, &amount, &kind, &payMonths)
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
	if kind == PaymentKindRenewal {
		return applyRenewal(ctx, tx, paymentID, rentalID, payMonths, amount)
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
		if err := insertPaymentRefundRequest(ctx, tx, paymentID, refundReasonBookingLost); err != nil {
			return fmt.Errorf("repository.ActivatePaidRental refund request: %w", err)
		}
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

// LockMinutes returns how long a booking is held in this ЖК before payment.
func (r *Repo) LockMinutes(ctx context.Context, jkID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT lock_minutes FROM jks WHERE id = $1`, jkID).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return n, err
}

func (r *Repo) SetLockMinutes(ctx context.Context, jkID string, minutes int) error {
	res, err := r.db.ExecContext(ctx, `UPDATE jks SET lock_minutes = $2 WHERE id = $1`, jkID, minutes)
	if err != nil {
		return fmt.Errorf("repository.SetLockMinutes: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

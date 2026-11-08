package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/kaekkr/kladovki/internal/models"
)

// remainingValueSQL: money for the not-yet-used part of a rental
// = total_paid × (time left / whole paid period). Alias of rentals must be "r".
const remainingValueSQL = `(r.total_paid * GREATEST(0, EXTRACT(EPOCH FROM (r.ends_at - now())))
	/ NULLIF(EXTRACT(EPOCH FROM (r.ends_at - r.starts_at)), 0))`

// newPriceSQL: monthly price of the storage at tariff $2 (alias "s").
const newPriceSQL = `ROUND(s.area * $2)::bigint`

// affectedByDecreaseSQL selects active rentals of JK $1 that get cheaper at tariff $2.
const affectedByDecreaseSQL = `
	FROM rentals r JOIN storages s ON s.id = r.storage_id
	WHERE r.jk_id = $1 AND r.status = 'active' AND r.ends_at > now()
	  AND ` + newPriceSQL + ` < r.price_per_month`

// TariffDecreasePreview: how many rentals are affected and how much money
// is owed back in total if the tariff drops to newTariff.
func (r *Repo) TariffDecreasePreview(ctx context.Context, jkID string, newTariff int64) (models.TariffPreviewResponse, error) {
	var out models.TariffPreviewResponse
	err := r.db.QueryRowContext(ctx, `
		SELECT count(*), COALESCE(SUM(ROUND(`+remainingValueSQL+` * (1 - `+newPriceSQL+`::numeric / r.price_per_month))), 0)::bigint
		`+affectedByDecreaseSQL, jkID, newTariff).Scan(&out.AffectedCount, &out.TotalDifference)
	if err != nil {
		return out, fmt.Errorf("repository.TariffDecreasePreview: %w", err)
	}
	return out, nil
}

// CreateTariffRecalcRequests asks every affected tenant to choose extend/refund.
// An unanswered request from an earlier change is superseded by the new one
// (its amount is recomputed against the latest tariff).
func (r *Repo) CreateTariffRecalcRequests(ctx context.Context, jkID string, newTariff int64) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE requests SET status = 'superseded', resolved_at = now()
		WHERE jk_id = $1 AND type = 'tariff_recalc' AND status = 'awaiting_choice'`, jkID); err != nil {
		return 0, fmt.Errorf("supersede: %w", err)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT r.id, r.user_id, r.price_per_month, `+newPriceSQL+`,
		       ROUND(`+remainingValueSQL+` * (1 - `+newPriceSQL+`::numeric / r.price_per_month))::bigint
		`+affectedByDecreaseSQL, jkID, newTariff)
	if err != nil {
		return 0, fmt.Errorf("select affected: %w", err)
	}
	type row struct {
		rentalID, userID   string
		oldPrice, newPrice int64
		amount             int64
	}
	var affected []row
	for rows.Next() {
		var x row
		if err := rows.Scan(&x.rentalID, &x.userID, &x.oldPrice, &x.newPrice, &x.amount); err != nil {
			rows.Close()
			return 0, err
		}
		affected = append(affected, x)
	}
	rows.Close()

	for _, x := range affected {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO requests (id, jk_id, user_id, rental_id, type, status, amount, old_price, new_price)
			VALUES ($1,$2,$3,$4,'tariff_recalc','awaiting_choice',$5,$6,$7)`,
			uuid.NewString(), jkID, x.userID, x.rentalID, x.amount, x.oldPrice, x.newPrice); err != nil {
			return 0, fmt.Errorf("insert request: %w", err)
		}
	}
	return int64(len(affected)), tx.Commit()
}

// ApplyTariffChoice records the tenant's answer to a tariff_recalc request.
//   - extend: remaining time is stretched by old/new price; applied at once.
//   - refund: the difference goes to the OSI as an open request.
func (r *Repo) ApplyTariffChoice(ctx context.Context, requestID, userID, choice string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var rentalID, status string
	var amount, newPrice int64
	err = tx.QueryRowContext(ctx, `
		SELECT COALESCE(rental_id, ''), status, amount, new_price FROM requests
		WHERE id = $1 AND user_id = $2 AND type = 'tariff_recalc'
		FOR UPDATE`, requestID, userID).Scan(&rentalID, &status, &amount, &newPrice)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if status != string(models.RequestAwaitingChoice) {
		return ErrConflict
	}

	switch choice {
	case "extend":
		_, err = tx.ExecContext(ctx, `
			UPDATE rentals r
			SET ends_at = r.ends_at + GREATEST(interval '0', r.ends_at - now()) * ((r.price_per_month::numeric / $2) - 1),
			    price_per_month = $2
			WHERE r.id = $1 AND r.status = 'active'`, rentalID, newPrice)
		if err == nil {
			_, err = tx.ExecContext(ctx, `
				UPDATE requests SET status = 'done', choice = 'extend', resolved_at = now() WHERE id = $1`, requestID)
		}
	case "refund":
		_, err = tx.ExecContext(ctx, `
			UPDATE rentals SET price_per_month = $2, total_paid = GREATEST(0, total_paid - $3)
			WHERE id = $1 AND status = 'active'`, rentalID, newPrice, amount)
		if err == nil {
			_, err = tx.ExecContext(ctx, `UPDATE requests SET status = 'open', choice = 'refund' WHERE id = $1`, requestID)
		}
	default:
		return ErrInvalid
	}
	if err != nil {
		return fmt.Errorf("repository.ApplyTariffChoice: %w", err)
	}
	return tx.Commit()
}

// CreateEarlyTermination files a tenant's request to end an active rental early.
func (r *Repo) CreateEarlyTermination(ctx context.Context, rentalID, userID, comment string) (*models.Request, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var jkID string
	var amount int64
	err = tx.QueryRowContext(ctx, `
		SELECT r.jk_id, ROUND(COALESCE(`+remainingValueSQL+`, 0))::bigint
		FROM rentals r
		WHERE r.id = $1 AND r.user_id = $2 AND r.status = 'active' AND r.ends_at > now()
		FOR UPDATE`, rentalID, userID).Scan(&jkID, &amount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var pending bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM requests WHERE rental_id = $1 AND type = 'early_termination' AND status = 'open')`,
		rentalID).Scan(&pending); err != nil {
		return nil, err
	}
	if pending {
		return nil, ErrConflict
	}

	id := uuid.NewString()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO requests (id, jk_id, user_id, rental_id, type, status, amount, comment)
		VALUES ($1,$2,$3,$4,'early_termination','open',$5,$6)`,
		id, jkID, userID, rentalID, amount, comment); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.getRequest(ctx, `rq.id = $1`, id)
}

// ResolveRequest is the OSI's decision on an open request in its JK.
// Approving an early termination ends the rental and frees the storage.
func (r *Repo) ResolveRequest(ctx context.Context, requestID, jkID, adminID, action, resolution string) error {
	if action != "approve" && action != "reject" {
		return ErrInvalid
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var typ, status, rentalID string
	err = tx.QueryRowContext(ctx, `
		SELECT type, status, COALESCE(rental_id, '') FROM requests
		WHERE id = $1 AND jk_id = $2 FOR UPDATE`, requestID, jkID).Scan(&typ, &status, &rentalID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if status != string(models.RequestOpen) {
		return ErrConflict
	}

	newStatus := models.RequestRejected
	if action == "approve" {
		newStatus = models.RequestDone
		if typ == string(models.RequestEarlyTermination) && rentalID != "" {
			var storageID string
			err := tx.QueryRowContext(ctx, `
				UPDATE rentals SET status = 'cancelled', ends_at = LEAST(ends_at, now())
				WHERE id = $1 AND status = 'active' RETURNING storage_id`, rentalID).Scan(&storageID)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			if storageID != "" {
				if _, err := tx.ExecContext(ctx, `
					UPDATE storages SET status = 'free', transfer_to_iin = NULL, transfer_until = NULL
					WHERE id = $1 AND status = 'occupied'`, storageID); err != nil {
					return err
				}
			}
		}
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE requests SET status = $2, resolution = $3, resolved_at = now(), resolved_by = $4
		WHERE id = $1`, requestID, newStatus, resolution, adminID); err != nil {
		return err
	}
	return tx.Commit()
}

// insertPaymentRefundRequest is used inside ActivatePaidRental's transaction.
func insertPaymentRefundRequest(ctx context.Context, tx *sql.Tx, paymentID, reason string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO requests (id, jk_id, user_id, rental_id, type, status, amount, comment)
		SELECT $1, r.jk_id, p.user_id, p.rental_id, 'payment_refund', 'open', p.amount, $3
		FROM payments p JOIN rentals r ON r.id = p.rental_id
		WHERE p.id = $2`, uuid.NewString(), paymentID, reason)
	return err
}

const requestSelect = `
	SELECT rq.id, rq.jk_id, rq.user_id, u.full_name, u.phone, COALESCE(rq.rental_id, ''),
	       COALESCE(s.number, ''), rq.type, rq.status, rq.amount, rq.choice,
	       rq.old_price, rq.new_price, rq.comment, rq.resolution, rq.created_at, rq.resolved_at
	FROM requests rq
	JOIN users u ON u.id = rq.user_id
	LEFT JOIN rentals r ON r.id = rq.rental_id
	LEFT JOIN storages s ON s.id = r.storage_id`

func scanRequest(s rowScanner) (*models.Request, error) {
	var q models.Request
	var resolved sql.NullTime
	err := s.Scan(&q.ID, &q.JKID, &q.UserID, &q.UserName, &q.UserPhone, &q.RentalID,
		&q.StorageNumber, &q.Type, &q.Status, &q.Amount, &q.Choice,
		&q.OldPrice, &q.NewPrice, &q.Comment, &q.Resolution, &q.CreatedAt, &resolved)
	if err != nil {
		return nil, err
	}
	if resolved.Valid {
		t := resolved.Time
		q.ResolvedAt = &t
	}
	return &q, nil
}

func (r *Repo) getRequest(ctx context.Context, where string, args ...any) (*models.Request, error) {
	q, err := scanRequest(r.db.QueryRowContext(ctx, requestSelect+" WHERE "+where, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return q, err
}

func (r *Repo) listRequests(ctx context.Context, where string, args ...any) ([]*models.Request, error) {
	rows, err := r.db.QueryContext(ctx, requestSelect+" WHERE "+where+`
		ORDER BY CASE rq.status WHEN 'open' THEN 0 WHEN 'awaiting_choice' THEN 1 ELSE 2 END, rq.created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []*models.Request{}
	for rows.Next() {
		q, err := scanRequest(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, q)
	}
	return list, rows.Err()
}

// ListRequestsByUser: everything addressed to / filed by the tenant (superseded ones hidden).
func (r *Repo) ListRequestsByUser(ctx context.Context, userID string) ([]*models.Request, error) {
	return r.listRequests(ctx, `rq.user_id = $1 AND rq.status <> 'superseded'`, userID)
}

// ListRequestsByJK: the OSI's request monitor.
func (r *Repo) ListRequestsByJK(ctx context.Context, jkID string) ([]*models.Request, error) {
	return r.listRequests(ctx, `rq.jk_id = $1 AND rq.status <> 'superseded'`, jkID)
}

// CountOpenRequests is used for the badge in the admin menu.
func (r *Repo) CountOpenRequests(ctx context.Context, jkID string) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM requests WHERE jk_id = $1 AND status = 'open'`, jkID).Scan(&n)
	return n, err
}

package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/kaekkr/kladovki/internal/models"
)

// remainingValueSQL: money for the not-yet-used part of a rental (alias "r").
// The unused paid time was worth V = value_left at A = value_at and runs down
// linearly to 0 at ends_at:  V × (ends_at − now) / (ends_at − A), clamped to [0, V].
// A fresh rental is never re-anchored (V = total_paid, A = starts_at), so an
// N-month rental at price P is worth exactly P×N at start and 0 at the end.
// Renewal, extend and refund re-anchor at the moment they happen, so later
// changes of ends_at / total_paid can't inflate it. See docs/domain.md.
const remainingValueSQL = `COALESCE(COALESCE(r.value_left, r.total_paid) * LEAST(1, GREATEST(0,
	EXTRACT(EPOCH FROM (r.ends_at - now()))
	/ NULLIF(EXTRACT(EPOCH FROM (r.ends_at - COALESCE(r.value_at, r.starts_at))), 0))), 0)`

// remainingValueRoundedSQL is remainingValueSQL in whole tenge.
const remainingValueRoundedSQL = `ROUND(` + remainingValueSQL + `)::bigint`

// newPriceSQL: monthly price of the storage at tariff $2 (alias "s").
const newPriceSQL = `ROUND(s.area * $2)::bigint`

// affectedByDecreaseSQL selects active rentals of JK $1 that get cheaper at tariff $2.
// A rental with an open early termination is left out: its whole remaining
// value is already claimed by the termination, a tariff refund would pay twice.
const affectedByDecreaseSQL = `
	FROM rentals r JOIN storages s ON s.id = r.storage_id
	WHERE r.jk_id = $1 AND r.status = 'active' AND r.ends_at > now()
	  AND ` + newPriceSQL + ` < r.price_per_month
	  AND NOT EXISTS (SELECT 1 FROM requests t
	      WHERE t.rental_id = r.id AND t.type = 'early_termination' AND t.status = 'open')`

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
	n, err := createTariffRecalcRequestsTx(ctx, tx, jkID, newTariff)
	if err != nil {
		return 0, err
	}
	return n, tx.Commit()
}

func createTariffRecalcRequestsTx(ctx context.Context, tx *sql.Tx, jkID string, newTariff int64) (int64, error) {
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
	return int64(len(affected)), nil
}

// ApplyTariffChoice records the tenant's answer to a tariff_recalc request.
//   - extend: remaining time is stretched by old/new price; applied at once.
//   - refund: the difference (computed now) goes to the OSI as an open request.
//     total_paid is lowered only when the ОСИ confirms the money was returned.
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

	// The rental may have ended since the request was created (early termination,
	// release, cancel): then there is nothing to extend or refund.
	var rentalStatus string
	err = tx.QueryRowContext(ctx, `SELECT status FROM rentals WHERE id = $1 FOR UPDATE`, rentalID).Scan(&rentalStatus)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if rentalStatus != string(models.RentalStatusActive) {
		_, _ = tx.ExecContext(ctx, `UPDATE requests SET status = 'superseded', resolved_at = now() WHERE id = $1`, requestID)
		_ = tx.Commit()
		return ErrConflict
	}
	// An open early termination already claims the whole remaining value.
	var terminating bool
	if err := tx.QueryRowContext(ctx, openTerminationSQL, rentalID).Scan(&terminating); err != nil {
		return err
	}
	if terminating {
		return ErrTerminationPending
	}

	switch choice {
	case "extend":
		_, err = tx.ExecContext(ctx, `
			UPDATE rentals r
			SET ends_at = r.ends_at + GREATEST(interval '0', r.ends_at - now()) * ((r.price_per_month::numeric / $2) - 1),
			    price_per_month = $2,
			    value_left = `+remainingValueRoundedSQL+`, value_at = now()
			WHERE r.id = $1 AND r.status = 'active'`, rentalID, newPrice)
		if err == nil {
			_, err = tx.ExecContext(ctx, `
				UPDATE requests SET status = 'done', choice = 'extend', resolved_at = now() WHERE id = $1`, requestID)
		}
	case "refund":
		// The difference for the time still left now, not when the tariff changed.
		err = tx.QueryRowContext(ctx, `
			SELECT GREATEST(0, ROUND(`+remainingValueSQL+` * (1 - $2::numeric / r.price_per_month)))::bigint
			FROM rentals r WHERE r.id = $1`, rentalID, newPrice).Scan(&amount)
		if err == nil {
			_, err = tx.ExecContext(ctx, `
				UPDATE rentals r SET price_per_month = $2,
				    value_left = GREATEST(0, `+remainingValueRoundedSQL+` - $3), value_at = now()
				WHERE r.id = $1 AND r.status = 'active'`, rentalID, newPrice, amount)
		}
		if err == nil {
			_, err = tx.ExecContext(ctx, `
				UPDATE requests SET status = 'open', choice = 'refund', amount = $2, refund_deducted = false
				WHERE id = $1`, requestID, amount)
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
		SELECT r.jk_id, `+remainingValueRoundedSQL+`
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
	if err := tx.QueryRowContext(ctx, openTerminationSQL, rentalID).Scan(&pending); err != nil {
		return nil, err
	}
	if pending {
		return nil, ErrConflict
	}
	// An unanswered tariff choice could still turn into a refund of part of
	// the same money: the tenant answers it first (refund lowers value_left,
	// so termination + refund never exceed the unused value).
	var choosing bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM requests
			WHERE rental_id = $1 AND type = 'tariff_recalc' AND status = 'awaiting_choice')`,
		rentalID).Scan(&choosing); err != nil {
		return nil, err
	}
	if choosing {
		return nil, ErrTariffChoicePending
	}
	// A renewal invoice just sent may still be paid: its money would change
	// the amount. The tenant waits for it (or for it to go stale).
	var renewing bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM payments
			WHERE rental_id = $1 AND kind = 'renewal' AND status = 'pending' AND created_at > now() - $2::interval)`,
		rentalID, fmt.Sprintf("%d seconds", int(RenewalPendingWindow.Seconds()))).Scan(&renewing); err != nil {
		return nil, err
	}
	if renewing {
		return nil, ErrRenewalPending
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
	var amount int64
	var deducted bool
	err = tx.QueryRowContext(ctx, `
		SELECT type, status, COALESCE(rental_id, ''), amount, refund_deducted FROM requests
		WHERE id = $1 AND jk_id = $2 FOR UPDATE`, requestID, jkID).Scan(&typ, &status, &rentalID, &amount, &deducted)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if status != string(models.RequestOpen) {
		return ErrConflict
	}
	// The tenant already gave up the extension for this refund: the ОСИ can
	// only confirm it was paid.
	if typ == string(models.RequestTariffRecalc) && action == "reject" {
		return ErrRefundNotRejectable
	}

	newStatus := models.RequestRejected
	if action == "approve" {
		newStatus = models.RequestDone
		// A legacy refund (refund_deducted) was taken off total_paid at choice time.
		if typ == string(models.RequestTariffRecalc) && rentalID != "" && !deducted {
			// The money is back with the tenant: «Оплачено» drops by it now.
			if _, err := tx.ExecContext(ctx, `
				UPDATE rentals SET total_paid = GREATEST(0, total_paid - $2) WHERE id = $1`, rentalID, amount); err != nil {
				return err
			}
		}
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
			if _, err := tx.ExecContext(ctx, supersedeRentalRequestsSQL, rentalID, requestID); err != nil {
				return err
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

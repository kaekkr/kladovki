package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/kaekkr/kladovki/internal/models"
	"github.com/kaekkr/kladovki/internal/notify"
)

// CreateNotification stores an in-app notification and queues one delivery
// per channel. With a dedupeKey the same notification is never created twice
// for a user; created=false then.
func (r *Repo) CreateNotification(ctx context.Context, userID, kind, title, body, link, dedupeKey string, channels []string) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	id := uuid.NewString()
	res, err := tx.ExecContext(ctx, `
		INSERT INTO notifications (id, user_id, kind, title, body, link, dedupe_key)
		VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,''))
		ON CONFLICT (user_id, dedupe_key) WHERE dedupe_key IS NOT NULL DO NOTHING`,
		id, userID, kind, title, body, link, dedupeKey)
	if err != nil {
		return false, fmt.Errorf("repository.CreateNotification: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return false, nil
	}
	for _, ch := range channels {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO notification_deliveries (id, notification_id, channel) VALUES ($1,$2,$3)`,
			uuid.NewString(), id, ch); err != nil {
			return false, fmt.Errorf("repository.CreateNotification delivery: %w", err)
		}
	}
	return true, tx.Commit()
}

func (r *Repo) ListNotifications(ctx context.Context, userID string, limit int) ([]*models.Notification, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, kind, title, body, link, read_at IS NOT NULL, created_at
		FROM notifications WHERE user_id = $1
		ORDER BY created_at DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []*models.Notification{}
	for rows.Next() {
		n := &models.Notification{}
		if err := rows.Scan(&n.ID, &n.Kind, &n.Title, &n.Body, &n.Link, &n.Read, &n.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, n)
	}
	return list, rows.Err()
}

func (r *Repo) UnreadNotifications(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID).Scan(&n)
	return n, err
}

// MarkNotificationsRead marks one notification (id != "") or all of the user's as read.
func (r *Repo) MarkNotificationsRead(ctx context.Context, userID, id string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE notifications SET read_at = now()
		WHERE user_id = $1 AND read_at IS NULL AND ($2 = '' OR id = $2)`, userID, id)
	return err
}

// PendingDelivery is a queued message for one channel.
type PendingDelivery struct {
	ID        string
	Channel   string
	Attempts  int
	Recipient notify.Recipient
	Message   notify.Message
}

// PendingDeliveries claims up to `limit` due deliveries for sending: they are
// leased for 5 minutes (SKIP LOCKED), so a second server instance never picks
// the same ones; FinishDelivery then records the outcome. If the sender dies,
// the lease runs out and the delivery is retried.
func (r *Repo) PendingDeliveries(ctx context.Context, limit int) ([]PendingDelivery, error) {
	rows, err := r.db.QueryContext(ctx, `
		WITH claimed AS (
			UPDATE notification_deliveries d
			SET next_attempt_at = now() + interval '5 minutes'
			WHERE d.id IN (
				SELECT id FROM notification_deliveries
				WHERE status = 'pending' AND next_attempt_at <= now()
				ORDER BY created_at
				LIMIT $1
				FOR UPDATE SKIP LOCKED)
			RETURNING d.id, d.channel, d.attempts, d.notification_id, d.created_at
		)
		SELECT c.id, c.channel, c.attempts, u.full_name, COALESCE(u.email, ''), u.phone,
		       COALESCE(u.telegram_chat_id, ''), n.title, n.body, n.link
		FROM claimed c
		JOIN notifications n ON n.id = c.notification_id
		JOIN users u ON u.id = n.user_id
		ORDER BY c.created_at`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PendingDelivery
	for rows.Next() {
		var d PendingDelivery
		if err := rows.Scan(&d.ID, &d.Channel, &d.Attempts, &d.Recipient.Name, &d.Recipient.Email,
			&d.Recipient.Phone, &d.Recipient.TelegramChatID, &d.Message.Title, &d.Message.Body, &d.Message.Link); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// FinishDelivery records the outcome. Failed attempts are retried with
// back-off until maxAttempts, then marked failed.
func (r *Repo) FinishDelivery(ctx context.Context, id string, sendErr error, skipped bool, maxAttempts int) error {
	switch {
	case skipped:
		_, err := r.db.ExecContext(ctx, `UPDATE notification_deliveries SET status = 'skipped' WHERE id = $1`, id)
		return err
	case sendErr == nil:
		_, err := r.db.ExecContext(ctx, `
			UPDATE notification_deliveries SET status = 'sent', sent_at = now(), attempts = attempts + 1
			WHERE id = $1`, id)
		return err
	default:
		msg := sendErr.Error()
		if len(msg) > 500 {
			msg = msg[:500]
		}
		_, err := r.db.ExecContext(ctx, `
			UPDATE notification_deliveries
			SET attempts = attempts + 1, last_error = $2,
			    status = CASE WHEN attempts + 1 >= $3 THEN 'failed' ELSE 'pending' END,
			    next_attempt_at = now() + (interval '1 minute' * power(2, attempts))
			WHERE id = $1`, id, msg, maxAttempts)
		return err
	}
}

// ---------- recipients ----------

func (r *Repo) ListJKAdminIDs(ctx context.Context, jkID string) ([]string, error) {
	return r.ids(ctx, `SELECT id FROM users WHERE jk_id = $1 AND role = 'admin'`, jkID)
}

func (r *Repo) ids(ctx context.Context, q string, args ...any) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// UserContacts returns which contact points a user has (for choosing channels).
func (r *Repo) UserContacts(ctx context.Context, userID string) (notify.Recipient, error) {
	var rc notify.Recipient
	err := r.db.QueryRowContext(ctx, `
		SELECT full_name, COALESCE(email, ''), phone, COALESCE(telegram_chat_id, '')
		FROM users WHERE id = $1`, userID).Scan(&rc.Name, &rc.Email, &rc.Phone, &rc.TelegramChatID)
	if errors.Is(err, sql.ErrNoRows) {
		return rc, ErrNotFound
	}
	return rc, err
}

// ---------- Telegram linking ----------

func (r *Repo) CreateTelegramLinkToken(ctx context.Context, userID string) (string, error) {
	token := uuid.NewString()
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM telegram_link_tokens WHERE user_id = $1 OR created_at < now() - interval '1 day';
	`, userID)
	if err != nil {
		return "", err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO telegram_link_tokens (token, user_id) VALUES ($1,$2)`, token, userID)
	return token, err
}

// LinkTelegram consumes a link token (valid 1 day) and stores the chat id.
func (r *Repo) LinkTelegram(ctx context.Context, token, chatID string) (string, error) {
	var userID string
	err := r.db.QueryRowContext(ctx, `
		DELETE FROM telegram_link_tokens WHERE token = $1 AND created_at > now() - interval '1 day'
		RETURNING user_id`, token).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	_, err = r.db.ExecContext(ctx, `UPDATE users SET telegram_chat_id = $2 WHERE id = $1`, userID, chatID)
	return userID, err
}

func (r *Repo) UnlinkTelegram(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET telegram_chat_id = NULL WHERE id = $1`, userID)
	return err
}

// ---------- reminder queries ----------

// ReminderRental is an active rental with what reminders need.
type ReminderRental struct {
	RentalID      string
	UserID        string
	JKID          string
	StorageNumber string
	EndsAt        time.Time
	PricePerMonth int64   // ActiveRentalsOfJK only
	Area          float64 // ActiveRentalsOfJK only
}

// RentalsForReminders returns active rentals ending before `until`
// (includes already overdue ones).
func (r *Repo) RentalsForReminders(ctx context.Context, until time.Time) ([]ReminderRental, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT r.id, r.user_id, r.jk_id, s.number, r.ends_at
		FROM rentals r JOIN storages s ON s.id = r.storage_id
		WHERE r.status = 'active' AND r.ends_at < $1`, until)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ReminderRental
	for rows.Next() {
		var x ReminderRental
		if err := rows.Scan(&x.RentalID, &x.UserID, &x.JKID, &x.StorageNumber, &x.EndsAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// RentalForNotice loads the fields used in payment / request notifications.
func (r *Repo) RentalForNotice(ctx context.Context, rentalID string) (ReminderRental, error) {
	var x ReminderRental
	err := r.db.QueryRowContext(ctx, `
		SELECT r.id, r.user_id, r.jk_id, s.number, r.ends_at
		FROM rentals r JOIN storages s ON s.id = r.storage_id WHERE r.id = $1`, rentalID).
		Scan(&x.RentalID, &x.UserID, &x.JKID, &x.StorageNumber, &x.EndsAt)
	return x, err
}

func (r *Repo) RentalIDForPayment(ctx context.Context, paymentID string) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `SELECT rental_id FROM payments WHERE id = $1`, paymentID).Scan(&id)
	return id, err
}

// ActiveRentalsOfJK lists active rentals for tariff-change notices.
func (r *Repo) ActiveRentalsOfJK(ctx context.Context, jkID string) ([]ReminderRental, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT r.id, r.user_id, r.jk_id, s.number, r.ends_at, r.price_per_month, s.area
		FROM rentals r JOIN storages s ON s.id = r.storage_id
		WHERE r.jk_id = $1 AND r.status = 'active' AND r.ends_at > now()`, jkID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ReminderRental
	for rows.Next() {
		var x ReminderRental
		if err := rows.Scan(&x.RentalID, &x.UserID, &x.JKID, &x.StorageNumber, &x.EndsAt, &x.PricePerMonth, &x.Area); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// GetRequestByID returns a request regardless of owner (internal use).
func (r *Repo) GetRequestByID(ctx context.Context, id string) (*models.Request, error) {
	return r.getRequest(ctx, `rq.id = $1`, id)
}

// AwaitingChoiceRequests returns the tariff_recalc requests a JK's tenants still have to answer.
func (r *Repo) AwaitingChoiceRequests(ctx context.Context, jkID string) ([]*models.Request, error) {
	return r.listRequests(ctx, `rq.jk_id = $1 AND rq.status = 'awaiting_choice'`, jkID)
}

package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/kaekkr/kladovki/internal/models"
)

func (r *Repo) SetTariff(ctx context.Context, jkID string, amount int64) error {
	now := time.Now()
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO tariff_history (id, jk_id, amount, changed_at) VALUES ($1, $2, $3, $4)`,
		newID(), jkID, amount, now,
	)
	if err != nil {
		return fmt.Errorf("repository.SetTariff history: %w", err)
	}
	res, err := r.db.ExecContext(
		ctx,
		`UPDATE tariffs SET amount = $1, created_at = $2 WHERE jk_id = $3`, amount, now, jkID,
	)
	if err != nil {
		return fmt.Errorf("repository.SetTariff update: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		_, err = r.db.ExecContext(
			ctx,
			`INSERT INTO tariffs (id, jk_id, amount, created_at) VALUES ($1, $2, $3, $4)`,
			newID(), jkID, amount, now,
		)
		if err != nil {
			return fmt.Errorf("repository.SetTariff insert: %w", err)
		}
	}
	return nil
}

func (r *Repo) GetTariff(ctx context.Context, jkID string) (int64, error) {
	var amount int64
	err := r.db.QueryRowContext(ctx, `SELECT amount FROM tariffs WHERE jk_id = $1`, jkID).Scan(&amount)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("repository.GetTariff: %w", err)
	}
	return amount, nil
}

func (r *Repo) TariffHistory(ctx context.Context, jkID string) ([]models.TariffHistory, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, jk_id, amount, changed_at FROM tariff_history
		WHERE jk_id = $1 ORDER BY changed_at DESC`, jkID)
	if err != nil {
		return nil, fmt.Errorf("repository.TariffHistory: %w", err)
	}
	defer rows.Close()

	var list []models.TariffHistory
	for rows.Next() {
		var h models.TariffHistory
		if err := rows.Scan(&h.ID, &h.JKID, &h.Amount, &h.ChangedAt); err != nil {
			return nil, err
		}
		list = append(list, h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository.TariffHistory rows: %w", err)
	}

	return list, nil
}

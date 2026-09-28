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

const tariffColumns = `id, jk_id, amount, created_at, updated_at`

func (r *Repo) GetTariffPreview(ctx context.Context, jkID string, newAmount int64) (models.TariffPreviewResponse, error) {
	// 1. Получаем текущий тариф
	currentTariff, err := r.GetTariff(ctx, jkID)
	if err != nil || currentTariff <= newAmount {
		return models.TariffPreviewResponse{AffectedCount: 0, TotalDifference: 0}, nil
	}

	// 2. Исправленный запрос с нумерацией параметров под PostgreSQL ($1, $2, $3)
	query := `
		SELECT COUNT(r.id), COALESCE(SUM(($1 - $2) * s.area), 0)
		FROM rentals r
		JOIN storages s ON r.storage_id = s.id
		WHERE r.jk_id = $3 AND r.status = 'active'
	`

	var count, diffSum int64
	err = r.db.QueryRowContext(ctx, query, currentTariff, newAmount, jkID).Scan(&count, &diffSum)
	if err != nil {
		return models.TariffPreviewResponse{}, fmt.Errorf("repository.GetTariffPreview: %w", err)
	}

	return models.TariffPreviewResponse{
		AffectedCount:   count,
		TotalDifference: diffSum,
	}, nil
}

func (r *Repo) SetTariff(ctx context.Context, jkID string, amount int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("repository.SetTariff begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC()

	// 1. Insert history record
	historyID := uuid.NewString()
	historyQuery := `
		INSERT INTO tariff_history (id, jk_id, amount, changed_at) 
		VALUES ($1, $2, $3, $4)
	`
	if _, err := tx.ExecContext(ctx, historyQuery, historyID, jkID, amount, now); err != nil {
		return fmt.Errorf("repository.SetTariff insert history: %w", err)
	}

	// 2. Upsert (insert or update on conflict) the active tariff
	tariffID := uuid.NewString()
	upsertQuery := `
		INSERT INTO tariffs (id, jk_id, amount, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $4)
		ON CONFLICT (jk_id) DO UPDATE SET
			amount = EXCLUDED.amount,
			updated_at = EXCLUDED.updated_at
	`
	if _, err := tx.ExecContext(ctx, upsertQuery, tariffID, jkID, amount, now); err != nil {
		return fmt.Errorf("repository.SetTariff upsert tariff: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("repository.SetTariff commit tx: %w", err)
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

func (r *Repo) GetTariffModel(ctx context.Context, jkID string) (*models.Tariff, error) {
	query := fmt.Sprintf(`SELECT %s FROM tariffs WHERE jk_id = $1`, tariffColumns)

	var t models.Tariff
	err := r.db.QueryRowContext(ctx, query, jkID).Scan(
		&t.ID,
		&t.JKID,
		&t.Amount,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.GetTariffModel: %w", err)
	}

	return &t, nil
}

func (r *Repo) TariffHistory(ctx context.Context, jkID string) ([]models.TariffHistory, error) {
	query := `
		SELECT id, jk_id, amount, changed_at 
		FROM tariff_history
		WHERE jk_id = $1 
		ORDER BY changed_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, jkID)
	if err != nil {
		return nil, fmt.Errorf("repository.TariffHistory query: %w", err)
	}
	defer rows.Close()

	var list []models.TariffHistory
	for rows.Next() {
		var h models.TariffHistory
		if err := rows.Scan(&h.ID, &h.JKID, &h.Amount, &h.ChangedAt); err != nil {
			return nil, fmt.Errorf("repository.TariffHistory scan: %w", err)
		}
		list = append(list, h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository.TariffHistory rows: %w", err)
	}

	return list, nil
}

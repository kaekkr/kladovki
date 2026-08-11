package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/kaekkr/kladovki/internal/models"
)

const storageColumns = `id, jk_id, number, area, floor, entrance, price, status, created_at`

func (r *Repo) GetStorage(ctx context.Context, id string) (*models.Storage, error) {
	return r.GetStorageByID(ctx, id)
}

func (r *Repo) CreateStorage(ctx context.Context, s *models.Storage) error {
	if s.ID == "" {
		s.ID = newID()
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}
	if s.Status == "" {
		s.Status = models.StatusFree
	}
	query := `
		INSERT INTO storages (id, jk_id, number, area, floor, entrance, price, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.db.ExecContext(
		ctx, query,
		s.ID, s.JKID, s.Number, s.Area, s.Floor, s.Entrance, s.Price, s.Status, s.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("repository.CreateStorage: %w", err)
	}
	return nil
}

func (r *Repo) UpdateStorageStatus(ctx context.Context, id string, status models.StorageStatus) error {
	_, err := r.db.ExecContext(ctx, `UPDATE storages SET status = $1 WHERE id = $2`, status, id)
	if err != nil {
		return fmt.Errorf("repository.UpdateStorageStatus: %w", err)
	}
	return nil
}

func (r *Repo) GetStorageByID(ctx context.Context, id string) (*models.Storage, error) {
	query := fmt.Sprintf(`SELECT %s FROM storages WHERE id = $1`, storageColumns)

	var s models.Storage
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&s.ID, &s.JKID, &s.Number, &s.Area, &s.Floor, &s.Entrance, &s.Price, &s.Status, &s.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.GetStorageByID: %w", err)
	}

	return &s, nil
}

func (r *Repo) ListStoragesByJK(ctx context.Context, jkID string) ([]*models.Storage, error) {
	query := fmt.Sprintf(`SELECT %s FROM storages WHERE jk_id = $1`, storageColumns)
	rows, err := r.db.QueryContext(ctx, query, jkID)
	if err != nil {
		return nil, fmt.Errorf("repository.ListStoragesByJK: %w", err)
	}
	defer rows.Close()

	var list []*models.Storage
	for rows.Next() {
		var s models.Storage
		if err := rows.Scan(
			&s.ID, &s.JKID, &s.Number, &s.Area, &s.Floor, &s.Entrance, &s.Price, &s.Status, &s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("repository.ListStoragesByJK scan: %w", err)
		}
		list = append(list, &s)
	}

	return list, nil
}

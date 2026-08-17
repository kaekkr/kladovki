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

const storageColumns = `id, jk_id, number, area, floor, entrance, status, created_at`

func (r *Repo) CreateStorage(ctx context.Context, s *models.Storage) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	if s.Status == "" {
		s.Status = models.StatusFree
	}

	query := `
		INSERT INTO storages (id, jk_id, number, area, floor, entrance, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at
	`

	now := time.Now().UTC()

	return r.db.QueryRowContext(
		ctx,
		query,
		s.ID,
		s.JKID,
		s.Number,
		s.Area,
		s.Floor,
		s.Entrance,
		s.Status,
		now,
	).Scan(&s.CreatedAt)
}

func (r *Repo) GetStorageByID(ctx context.Context, id string) (*models.Storage, error) {
	query := fmt.Sprintf(`SELECT %s FROM storages WHERE id = $1`, storageColumns)
	return r.scanStorage(r.db.QueryRowContext(ctx, query, id))
}

func (r *Repo) GetStorage(ctx context.Context, id string) (*models.Storage, error) {
	return r.GetStorageByID(ctx, id)
}

func (r *Repo) ListStoragesByJK(ctx context.Context, jkID string) ([]*models.Storage, error) {
	query := fmt.Sprintf(`SELECT %s FROM storages WHERE jk_id = $1 ORDER BY number ASC`, storageColumns)
	rows, err := r.db.QueryContext(ctx, query, jkID)
	if err != nil {
		return nil, fmt.Errorf("repository.ListStoragesByJK: %w", err)
	}
	defer rows.Close()

	var list []*models.Storage
	for rows.Next() {
		s, err := r.scanStorage(rows)
		if err != nil {
			return nil, fmt.Errorf("repository.ListStoragesByJK scan: %w", err)
		}
		list = append(list, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository.ListStoragesByJK rows: %w", err)
	}

	return list, nil
}

func (r *Repo) GetAvailableStoragesByJK(ctx context.Context, jkID string) ([]*models.Storage, error) {
	// Dynamically check if a lock has expired in the rentals table.
	// If a lock expired (r.id IS NULL), treat the returned status as 'free'.
	query := `
		SELECT 
			s.id, 
			s.jk_id, 
			s.number, 
			s.area, 
			s.floor, 
			s.entrance, 
			CASE 
				WHEN s.status = 'locked' AND r.id IS NULL THEN 'free'
				ELSE s.status 
			END AS status, 
			s.created_at
		FROM storages s
		LEFT JOIN rentals r ON s.id = r.storage_id AND r.status = 'locked' AND r.ends_at > NOW()
		WHERE s.jk_id = $1 
		  AND s.status != 'occupied'
		  AND r.id IS NULL
		ORDER BY s.number ASC
	`

	rows, err := r.db.QueryContext(ctx, query, jkID)
	if err != nil {
		return nil, fmt.Errorf("repository.GetAvailableStoragesByJK: %w", err)
	}
	defer rows.Close()

	var list []*models.Storage
	for rows.Next() {
		s, err := r.scanStorage(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository.GetAvailableStoragesByJK rows: %w", err)
	}

	return list, nil
}

func (r *Repo) UpdateStorage(ctx context.Context, s *models.Storage) error {
	query := `
		UPDATE storages 
		SET number = $1, area = $2, floor = $3, entrance = $4, status = $5
		WHERE id = $6
	`

	res, err := r.db.ExecContext(ctx, query, s.Number, s.Area, s.Floor, s.Entrance, s.Status, s.ID)
	if err != nil {
		return fmt.Errorf("repository.UpdateStorage: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("repository.UpdateStorage rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Repo) UpdateStorageStatus(ctx context.Context, id string, status models.StorageStatus) error {
	res, err := r.db.ExecContext(ctx, `UPDATE storages SET status = $1 WHERE id = $2`, status, id)
	if err != nil {
		return fmt.Errorf("repository.UpdateStorageStatus: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("repository.UpdateStorageStatus rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Repo) DeleteStorage(ctx context.Context, id string) error {
	query := `DELETE FROM storages WHERE id = $1`

	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("repository.DeleteStorage: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("repository.DeleteStorage rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Repo) scanStorage(s rowScanner) (*models.Storage, error) {
	var st models.Storage

	err := s.Scan(
		&st.ID,
		&st.JKID,
		&st.Number,
		&st.Area,
		&st.Floor,
		&st.Entrance,
		&st.Status,
		&st.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &st, nil
}

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

const jkColumns = `id, name, created_at`

func (r *Repo) CreateJK(ctx context.Context, jk *models.JK) error {
	if jk.ID == "" {
		jk.ID = uuid.NewString()
	}

	query := `
		INSERT INTO jks (id, name, created_at)
		VALUES ($1, $2, $3)
		RETURNING created_at
	`

	now := time.Now().UTC()

	return r.db.QueryRowContext(
		ctx,
		query,
		jk.ID,
		jk.Name,
		now,
	).Scan(&jk.CreatedAt)
}

func (r *Repo) GetJKByID(ctx context.Context, id string) (*models.JK, error) {
	query := fmt.Sprintf(`SELECT %s FROM jks WHERE id = $1`, jkColumns)
	return r.scanJK(r.db.QueryRowContext(ctx, query, id))
}

func (r *Repo) GetJKByName(ctx context.Context, name string) (*models.JK, error) {
	query := fmt.Sprintf(`SELECT %s FROM jks WHERE name = $1`, jkColumns)
	return r.scanJK(r.db.QueryRowContext(ctx, query, name))
}

func (r *Repo) ListJKs(ctx context.Context) ([]*models.JK, error) {
	query := fmt.Sprintf(`SELECT %s FROM jks ORDER BY created_at DESC`, jkColumns)
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("repository.ListJKs query: %w", err)
	}
	defer rows.Close()

	var jks []*models.JK
	for rows.Next() {
		jk, err := r.scanJK(rows)
		if err != nil {
			return nil, fmt.Errorf("repository.ListJKs scan: %w", err)
		}
		jks = append(jks, jk)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository.ListJKs rows: %w", err)
	}

	return jks, nil
}

func (r *Repo) scanJK(s rowScanner) (*models.JK, error) {
	var jk models.JK

	err := s.Scan(
		&jk.ID,
		&jk.Name,
		&jk.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &jk, nil
}

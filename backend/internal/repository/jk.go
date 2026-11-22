package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/kaekkr/kladovki/internal/models"
)

const jkColumns = `id, name, address, owner_id, created_at, cadastral_number`

func (r *Repo) GetJKByID(ctx context.Context, id string) (*models.JK, error) {
	query := fmt.Sprintf(`SELECT %s FROM jks WHERE id = $1`, jkColumns)
	return r.scanJK(r.db.QueryRowContext(ctx, query, id))
}

func (r *Repo) GetJKByName(ctx context.Context, name string) (*models.JK, error) {
	query := fmt.Sprintf(`SELECT %s FROM jks WHERE name = $1`, jkColumns)
	return r.scanJK(r.db.QueryRowContext(ctx, query, name))
}

// GetJKByCadastral finds the ЖК whose cadastral number is exactly number.
func (r *Repo) GetJKByCadastral(ctx context.Context, number string) (*models.JK, error) {
	if number == "" {
		return nil, ErrNotFound
	}
	query := fmt.Sprintf(`SELECT %s FROM jks WHERE cadastral_number = $1`, jkColumns)
	return r.scanJK(r.db.QueryRowContext(ctx, query, number))
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
		&jk.Address,
		&jk.OwnerID,
		&jk.CreatedAt,
		&jk.CadastralNumber,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &jk, nil
}

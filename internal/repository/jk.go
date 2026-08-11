package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/kaekkr/kladovki/internal/models"
)

const jkColumns = `id, name, bin, contact, phone, email, owner_id, address, created_at`

func (r *Repo) CreateJK(ctx context.Context, jk *models.JK) error {
	if jk.ID == "" {
		jk.ID = newID()
	}
	if jk.CreatedAt.IsZero() {
		jk.CreatedAt = time.Now()
	}
	query := `
		INSERT INTO jks (id, name, bin, contact, phone, email, owner_id, address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := r.db.ExecContext(
		ctx, query,
		jk.ID, jk.Name, jk.BIN, jk.Contact, jk.Phone, jk.Email, jk.OwnerID, jk.Address, jk.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("repository.CreateJK: %w", err)
	}
	return nil
}

func (r *Repo) GetJK(ctx context.Context, id string) (*models.JK, error) {
	return r.GetJKByID(ctx, id)
}

func (r *Repo) GetJKByName(ctx context.Context, name string) (*models.JK, error) {
	query := fmt.Sprintf(`SELECT %s FROM jks WHERE name = $1`, jkColumns)
	var jk models.JK
	err := r.db.QueryRowContext(ctx, query, name).Scan(
		&jk.ID, &jk.Name, &jk.BIN, &jk.Contact, &jk.Phone, &jk.Email, &jk.OwnerID, &jk.Address, &jk.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.GetJKByName: %w", err)
	}
	return &jk, nil
}

func (r *Repo) GetJKByID(ctx context.Context, id string) (*models.JK, error) {
	query := fmt.Sprintf(`SELECT %s FROM jks WHERE id = $1`, jkColumns)

	var jk models.JK
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&jk.ID, &jk.Name, &jk.BIN, &jk.Contact, &jk.Phone, &jk.Email, &jk.OwnerID, &jk.Address, &jk.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.GetJKByID: %w", err)
	}

	return &jk, nil
}

func (r *Repo) ListJKs(ctx context.Context) ([]*models.JK, error) {
	query := `SELECT id, name, bin, contact, phone, email, owner_id, address, created_at FROM jks`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("repository.ListJKs query: %w", err)
	}
	defer rows.Close()

	var jks []*models.JK
	for rows.Next() {
		var jk models.JK
		var contact, email, ownerID, address sql.NullString

		if err := rows.Scan(
			&jk.ID,
			&jk.Name,
			&jk.BIN,
			&contact,
			&jk.Phone,
			&email,
			&ownerID, // 👈 Scans NULL safely
			&address,
			&jk.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("repository.ListJKs scan: %w", err)
		}

		jk.Contact = contact.String
		jk.Email = email.String
		jk.OwnerID = ownerID.String
		jk.Address = address.String

		jks = append(jks, &jk)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository.ListJKs rows: %w", err)
	}

	return jks, nil
}

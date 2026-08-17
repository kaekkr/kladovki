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

const userColumns = `id, full_name, phone, email, password_hash, role, jk_id, created_at`

func (r *Repo) CreateUser(ctx context.Context, u *models.User) error {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}

	query := `
		INSERT INTO users (id, full_name, phone, email, password_hash, role, jk_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at
	`

	now := time.Now().UTC()

	return r.db.QueryRowContext(
		ctx,
		query,
		u.ID,
		u.FullName,
		u.Phone,
		nullString(u.Email),
		u.PasswordHash,
		u.Role,
		nullStringPtr(u.JKID),
		now,
	).Scan(&u.CreatedAt)
}

func (r *Repo) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	query := fmt.Sprintf(`SELECT %s FROM users WHERE id = $1`, userColumns)
	return r.scanUser(r.db.QueryRowContext(ctx, query, id))
}

func (r *Repo) GetUserByPhone(ctx context.Context, phone string) (*models.User, error) {
	query := fmt.Sprintf(`SELECT %s FROM users WHERE phone = $1`, userColumns)
	return r.scanUser(r.db.QueryRowContext(ctx, query, phone))
}

func (r *Repo) GetUserByLogin(ctx context.Context, login string) (*models.User, error) {
	query := fmt.Sprintf(`
		SELECT %s FROM users 
		WHERE phone = $1 OR (email IS NOT NULL AND email = $1)
	`, userColumns)
	return r.scanUser(r.db.QueryRowContext(ctx, query, login))
}

func (r *Repo) scanUser(s rowScanner) (*models.User, error) {
	var u models.User
	var email, jkID sql.NullString

	err := s.Scan(
		&u.ID,
		&u.FullName,
		&u.Phone,
		&email,
		&u.PasswordHash,
		&u.Role,
		&jkID,
		&u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if email.Valid {
		u.Email = email.String
	}
	if jkID.Valid {
		u.JKID = &jkID.String
	}

	return &u, nil
}

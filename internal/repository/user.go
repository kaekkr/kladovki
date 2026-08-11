package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/kaekkr/kladovki/internal/models"
)

const userColumns = `id, role, full_name, phone, email, iin, bin, password_hash, jk_id, created_at`

func (r *Repo) CreateUser(ctx context.Context, u *models.User) error {
	if u.ID == "" {
		u.ID = newID()
	}
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO users (id, role, full_name, phone, email, iin, bin, password_hash, jk_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.ExecContext(
		ctx, query,
		u.ID, u.Role, u.FullName, u.Phone, u.Email, u.IIN, u.BIN, u.PasswordHash, u.JKID, u.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("repository.CreateUser: %w", err)
	}

	return nil
}

func (r *Repo) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	query := fmt.Sprintf(`SELECT %s FROM users WHERE id = $1`, userColumns)
	user, err := r.scanUser(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.GetUserByID: %w", err)
	}
	return user, nil
}

func (r *Repo) GetUserByPhone(ctx context.Context, phone string) (*models.User, error) {
	query := fmt.Sprintf(`SELECT %s FROM users WHERE phone = $1`, userColumns)
	user, err := r.scanUser(r.db.QueryRowContext(ctx, query, phone))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.GetUserByPhone: %w", err)
	}
	return user, nil
}

func (r *Repo) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := fmt.Sprintf(`SELECT %s FROM users WHERE email = $1`, userColumns)
	user, err := r.scanUser(r.db.QueryRowContext(ctx, query, email))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.GetUserByEmail: %w", err)
	}
	return user, nil
}

func (r *Repo) UpdateUserJKID(ctx context.Context, userID, jkID string) error {
	query := `UPDATE users SET jk_id = $1 WHERE id = $2`
	res, err := r.db.ExecContext(ctx, query, jkID, userID)
	if err != nil {
		return fmt.Errorf("repository.UpdateUserJKID: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("repository.UpdateUserJKID: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Repo) scanUser(s rowScanner) (*models.User, error) {
	var u models.User
	err := s.Scan(
		&u.ID, &u.Role, &u.FullName, &u.Phone, &u.Email,
		&u.IIN, &u.BIN, &u.PasswordHash, &u.JKID, &u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

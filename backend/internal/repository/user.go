package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/kaekkr/kladovki/internal/models"
	"github.com/lib/pq"
)

const userColumns = `id, roles, full_name, phone, email, iin, bin, password_hash, jk_id, created_at`

func (r *Repo) DB() *sql.DB {
	return r.db
}

func (r *Repo) CreateUser(ctx context.Context, u *models.User) error {
	if u.ID == "" {
		u.ID = newID()
	}
	if u.CreatedAt.IsZero() {
		u.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO users (id, roles, full_name, phone, email, iin, bin, password_hash, jk_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.ExecContext(
		ctx, query,
		u.ID, pq.Array(u.Roles), u.FullName, u.Phone, u.Email, u.IIN, u.BIN, u.PasswordHash, u.JKID, u.CreatedAt,
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

func (r *Repo) GetUserByIIN(ctx context.Context, iin string) (*models.User, error) {
	query := fmt.Sprintf(`SELECT %s FROM users WHERE iin = $1`, userColumns)
	return r.scanUser(r.db.QueryRowContext(ctx, query, iin))
}

func (r *Repo) GetUserByBIN(ctx context.Context, bin string) (*models.User, error) {
	query := fmt.Sprintf(`SELECT %s FROM users WHERE bin = $1`, userColumns)
	return r.scanUser(r.db.QueryRowContext(ctx, query, bin))
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

func (r *Repo) UpdateUserRoles(ctx context.Context, userID string, roles []models.Role) error {
	query := `UPDATE users SET roles = $1 WHERE id = $2`
	res, err := r.db.ExecContext(ctx, query, pq.Array(roles), userID)
	if err != nil {
		return fmt.Errorf("repository.UpdateUserRoles: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("repository.UpdateUserRoles: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func (r *Repo) scanUser(s rowScanner) (*models.User, error) {
	var u models.User
	var roles []string

	err := s.Scan(
		&u.ID, pq.Array(&roles), &u.FullName, &u.Phone, &u.Email,
		&u.IIN, &u.BIN, &u.PasswordHash, &u.JKID, &u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	u.Roles = make([]models.Role, len(roles))
	for i, r := range roles {
		u.Roles[i] = models.Role(r)
	}

	return &u, nil
}

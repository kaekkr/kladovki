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

const userColumns = `id, full_name, phone, email, password_hash, role, jk_id, created_at, iin,
	status, review_comment, owner_type, premises_kind, premises_number, company_bin, company_name, must_change_password`

func (r *Repo) CreateUser(ctx context.Context, u *models.User) error {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}

	query := `
    INSERT INTO users (
        id, full_name, phone, email, password_hash,
        role, jk_id, created_at, iin,
        status, owner_type, premises_kind, premises_number, company_bin, company_name
    )
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
    RETURNING created_at
	`
	now := time.Now().UTC()
	if u.Status == "" {
		u.Status = models.UserActive
	}
	if u.OwnerType == "" {
		u.OwnerType = models.OwnerPerson
	}
	if u.PremisesKind == "" {
		u.PremisesKind = models.PremisesFlat
	}

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
		u.IIN,
		u.Status, u.OwnerType, u.PremisesKind, u.PremisesNumber, u.CompanyBIN, u.CompanyName,
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
		WHERE phone = $1 OR (email IS NOT NULL AND lower(email) = lower($1)) OR (iin IS NOT NULL AND iin = $1)
	`, userColumns)
	return r.scanUser(r.db.QueryRowContext(ctx, query, login))
}

func (r *Repo) scanUser(s rowScanner) (*models.User, error) {
	var u models.User
	var email, jkID, iin sql.NullString

	err := s.Scan(
		&u.ID,
		&u.FullName,
		&u.Phone,
		&email,
		&u.PasswordHash,
		&u.Role,
		&jkID,
		&u.CreatedAt,
		&iin,
		&u.Status, &u.ReviewComment, &u.OwnerType, &u.PremisesKind, &u.PremisesNumber, &u.CompanyBIN, &u.CompanyName,
		&u.MustChangePassword,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if iin.Valid {
		u.IIN = iin.String
	}

	if email.Valid {
		u.Email = email.String
	}
	if jkID.Valid {
		u.JKID = &jkID.String
	}

	return &u, nil
}

func (r *Repo) IINRegistered(ctx context.Context, iin string) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE iin = $1)`, iin).Scan(&ok)
	return ok, err
}

func (r *Repo) PhoneRegistered(ctx context.Context, phone string) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE phone = $1)`, phone).Scan(&ok)
	return ok, err
}

// ListResidents returns residents of a ЖК, optionally filtered by status, newest first.
func (r *Repo) ListResidents(ctx context.Context, jkID, status string) ([]*models.User, error) {
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT %s FROM users
		WHERE jk_id = $1 AND role = 'resident' AND ($2 = '' OR status = $2)
		ORDER BY created_at DESC`, userColumns), jkID, status)
	if err != nil {
		return nil, fmt.Errorf("repository.ListResidents: %w", err)
	}
	defer rows.Close()
	list := []*models.User{}
	for rows.Next() {
		u, err := r.scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("repository.ListResidents scan: %w", err)
		}
		list = append(list, u)
	}
	return list, rows.Err()
}

func (r *Repo) CountPendingResidents(ctx context.Context, jkID string) (int64, error) {
	var n int64
	err := r.db.QueryRowContext(ctx, `
		SELECT count(*) FROM users WHERE jk_id = $1 AND role = 'resident' AND status = 'pending'`, jkID).Scan(&n)
	return n, err
}

// SetResidentStatus is the ОСИ decision on a resident of their own ЖК.
func (r *Repo) SetResidentStatus(ctx context.Context, userID, jkID string, status models.UserStatus, comment, adminID string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE users SET status = $3, review_comment = $4, reviewed_by = $5, reviewed_at = now()
		WHERE id = $1 AND jk_id = $2 AND role = 'resident'`, userID, jkID, status, comment, adminID)
	if err != nil {
		return fmt.Errorf("repository.SetResidentStatus: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ResubmitVerification updates what a pending/rejected user declares and puts them back in the queue.
func (r *Repo) ResubmitVerification(ctx context.Context, userID string, v models.Verification) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE users SET status = 'pending', review_comment = '',
		       owner_type = $2, premises_kind = $3, premises_number = $4, company_bin = $5, company_name = $6
		WHERE id = $1 AND role = 'resident' AND status IN ('pending', 'rejected')`,
		userID, v.OwnerType, v.PremisesKind, v.PremisesNumber, v.CompanyBIN, v.CompanyName)
	if err != nil {
		return fmt.Errorf("repository.ResubmitVerification: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrLocked
	}
	return nil
}

// SetPassword stores a new password hash; mustChange forces a change at next login.
func (r *Repo) SetPassword(ctx context.Context, userID, hash string, mustChange bool) error {
	res, err := r.db.ExecContext(ctx, `UPDATE users SET password_hash = $2, must_change_password = $3 WHERE id = $1`,
		userID, hash, mustChange)
	if err != nil {
		return fmt.Errorf("repository.SetPassword: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ResetResidentPassword sets a temporary password for a resident of the given ЖК.
func (r *Repo) ResetResidentPassword(ctx context.Context, userID, jkID, hash string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE users SET password_hash = $3, must_change_password = true
		WHERE id = $1 AND jk_id = $2 AND role = 'resident'`, userID, jkID, hash)
	if err != nil {
		return fmt.Errorf("repository.ResetResidentPassword: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

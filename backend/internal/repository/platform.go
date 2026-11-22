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

// Uniqueness conflicts the platform operator can fix in the form.
var (
	ErrEmailTaken     = errors.New("email already used")
	ErrPhoneTaken     = errors.New("phone already used")
	ErrCadastralTaken = errors.New("cadastral number already used")
)

const jkSummaryQuery = `
	SELECT j.id, j.name, j.address, j.owner_id, j.created_at, j.cadastral_number,
	       (SELECT count(*) FROM storages s WHERE s.jk_id = j.id),
	       (SELECT count(*) FROM users u WHERE u.jk_id = j.id AND u.role = 'admin'),
	       (SELECT count(*) FROM users u WHERE u.jk_id = j.id AND u.role = 'resident')
	FROM jks j`

func scanJKSummary(s rowScanner) (*models.JKSummary, error) {
	var j models.JKSummary
	err := s.Scan(&j.ID, &j.Name, &j.Address, &j.OwnerID, &j.CreatedAt, &j.CadastralNumber,
		&j.StoragesCount, &j.AdminsCount, &j.ResidentsCount)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// ListJKSummaries returns every ЖК with its storage, ОСИ and resident counts, newest first.
func (r *Repo) ListJKSummaries(ctx context.Context) ([]*models.JKSummary, error) {
	rows, err := r.db.QueryContext(ctx, jkSummaryQuery+` ORDER BY j.created_at DESC, j.id`)
	if err != nil {
		return nil, fmt.Errorf("repository.ListJKSummaries: %w", err)
	}
	defer rows.Close()
	list := []*models.JKSummary{}
	for rows.Next() {
		j, err := scanJKSummary(rows)
		if err != nil {
			return nil, fmt.Errorf("repository.ListJKSummaries scan: %w", err)
		}
		list = append(list, j)
	}
	return list, rows.Err()
}

// GetJKSummary returns one ЖК with its counts.
func (r *Repo) GetJKSummary(ctx context.Context, id string) (*models.JKSummary, error) {
	return scanJKSummary(r.db.QueryRowContext(ctx, jkSummaryQuery+` WHERE j.id = $1`, id))
}

// CreateJKWithAdmin creates a ЖК and its first ОСИ in one transaction.
// jks.owner_id must point at an existing user, so the admin is inserted
// first (without a ЖК), then the ЖК, then the admin is attached to it.
func (r *Repo) CreateJKWithAdmin(ctx context.Context, jk *models.JK, admin *models.User) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("repository.CreateJKWithAdmin begin: %w", err)
	}
	defer tx.Rollback()

	if err := lockAccountCreation(ctx, tx); err != nil {
		return err
	}
	if err := checkCadastralFree(ctx, tx, jk.CadastralNumber, ""); err != nil {
		return err
	}
	if err := insertUserTx(ctx, tx, admin, models.RoleAdmin, nil); err != nil {
		return err
	}
	jk.ID = uuid.NewString()
	jk.OwnerID = admin.ID
	jk.CreatedAt = time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO jks (id, name, address, owner_id, created_at, cadastral_number)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		jk.ID, jk.Name, jk.Address, jk.OwnerID, jk.CreatedAt, jk.CadastralNumber); err != nil {
		return fmt.Errorf("repository.CreateJKWithAdmin insert jk: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE users SET jk_id = $2 WHERE id = $1`, admin.ID, jk.ID); err != nil {
		return fmt.Errorf("repository.CreateJKWithAdmin attach admin: %w", err)
	}
	jkID := jk.ID
	admin.JKID = &jkID
	return tx.Commit()
}

// UpdateJKDetails changes the name, address and cadastral number of a ЖК.
func (r *Repo) UpdateJKDetails(ctx context.Context, jk *models.JK) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("repository.UpdateJKDetails begin: %w", err)
	}
	defer tx.Rollback()

	if err := lockAccountCreation(ctx, tx); err != nil {
		return err
	}
	if err := checkCadastralFree(ctx, tx, jk.CadastralNumber, jk.ID); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE jks SET name = $2, address = $3, cadastral_number = $4 WHERE id = $1`,
		jk.ID, jk.Name, jk.Address, jk.CadastralNumber)
	if err != nil {
		return fmt.Errorf("repository.UpdateJKDetails: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return tx.Commit()
}

// AddJKAdmin creates another ОСИ account for an existing ЖК.
func (r *Repo) AddJKAdmin(ctx context.Context, jkID string, admin *models.User) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("repository.AddJKAdmin begin: %w", err)
	}
	defer tx.Rollback()

	var id string
	err = tx.QueryRowContext(ctx, `SELECT id FROM jks WHERE id = $1 FOR SHARE`, jkID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("repository.AddJKAdmin jk: %w", err)
	}
	if err := lockAccountCreation(ctx, tx); err != nil {
		return err
	}
	if err := insertUserTx(ctx, tx, admin, models.RoleAdmin, &id); err != nil {
		return err
	}
	return tx.Commit()
}

// ListJKAdmins returns the ОСИ accounts of a ЖК, oldest first.
func (r *Repo) ListJKAdmins(ctx context.Context, jkID string) ([]*models.User, error) {
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT %s FROM users WHERE jk_id = $1 AND role = 'admin' ORDER BY created_at, id`, userColumns), jkID)
	if err != nil {
		return nil, fmt.Errorf("repository.ListJKAdmins: %w", err)
	}
	defer rows.Close()
	list := []*models.User{}
	for rows.Next() {
		u, err := r.scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("repository.ListJKAdmins scan: %w", err)
		}
		list = append(list, u)
	}
	return list, rows.Err()
}

// ResetAdminPassword sets a temporary password for an ОСИ account (role admin only).
func (r *Repo) ResetAdminPassword(ctx context.Context, userID, hash string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE users SET password_hash = $2, must_change_password = true
		WHERE id = $1 AND role = 'admin'`, userID, hash)
	if err != nil {
		return fmt.Errorf("repository.ResetAdminPassword: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// GetUserByEmail finds a user by email, ignoring case.
func (r *Repo) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := fmt.Sprintf(`SELECT %s FROM users WHERE email IS NOT NULL AND lower(email) = lower($1)
		ORDER BY created_at LIMIT 1`, userColumns)
	return r.scanUser(r.db.QueryRowContext(ctx, query, email))
}

// CreateSuperadmin inserts a platform operator (no ЖК) unless the email or phone is taken.
func (r *Repo) CreateSuperadmin(ctx context.Context, u *models.User) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("repository.CreateSuperadmin begin: %w", err)
	}
	defer tx.Rollback()
	if err := lockAccountCreation(ctx, tx); err != nil {
		return err
	}
	if err := insertUserTx(ctx, tx, u, models.RoleSuperadmin, nil); err != nil {
		return err
	}
	return tx.Commit()
}

// lockAccountCreation serialises platform account/ЖК creation for the rest of
// the transaction: users.email has no unique index, so the "is it free?"
// check and the insert must not interleave with another operator's.
func lockAccountCreation(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('kladovki:platform-accounts'))`); err != nil {
		return fmt.Errorf("repository.lockAccountCreation: %w", err)
	}
	return nil
}

// insertUserTx checks email (case-insensitive) and phone uniqueness, then inserts an active user.
func insertUserTx(ctx context.Context, tx *sql.Tx, u *models.User, role models.Role, jkID *string) error {
	var emailTaken, phoneTaken bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM users WHERE email IS NOT NULL AND lower(email) = lower($1)),
		       EXISTS (SELECT 1 FROM users WHERE phone = $2)`, u.Email, u.Phone).Scan(&emailTaken, &phoneTaken); err != nil {
		return fmt.Errorf("repository.insertUser check: %w", err)
	}
	if emailTaken {
		return ErrEmailTaken
	}
	if phoneTaken {
		return ErrPhoneTaken
	}
	u.ID = uuid.NewString()
	u.Role = role
	u.Status = models.UserActive
	u.JKID = jkID
	err := tx.QueryRowContext(ctx, `
		INSERT INTO users (id, full_name, phone, email, password_hash, role, jk_id, created_at, status, must_change_password)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now(), 'active', $8)
		RETURNING created_at`,
		u.ID, u.FullName, u.Phone, u.Email, u.PasswordHash, u.Role, nullStringPtr(jkID), u.MustChangePassword,
	).Scan(&u.CreatedAt)
	if err != nil {
		return fmt.Errorf("repository.insertUser: %w", err)
	}
	return nil
}

// checkCadastralFree fails with ErrCadastralTaken when another ЖК (not exceptID) has the number.
func checkCadastralFree(ctx context.Context, tx *sql.Tx, number, exceptID string) error {
	if number == "" {
		return nil
	}
	var taken bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM jks WHERE cadastral_number = $1 AND id <> $2)`, number, exceptID).Scan(&taken); err != nil {
		return fmt.Errorf("repository.checkCadastralFree: %w", err)
	}
	if taken {
		return ErrCadastralTaken
	}
	return nil
}

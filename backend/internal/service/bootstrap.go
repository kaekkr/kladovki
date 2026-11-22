package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/kaekkr/kladovki/internal/models"
)

// SuperadminResult says what SetSuperadmin did.
type SuperadminResult string

const (
	SuperadminCreated       SuperadminResult = "created"
	SuperadminPasswordReset SuperadminResult = "password_reset"
	// SuperadminConflict: the email belongs to a user with another role; nothing was changed.
	SuperadminConflict SuperadminResult = "conflict"
)

// SetSuperadmin creates a platform operator with this email, or sets a new
// password for an existing one (other sessions end). Used by the one-off
// command `go run ./cmd/superadmin`, so no password lives in env vars.
// A user with another role is never promoted. The password is never logged.
func (s *Service) SetSuperadmin(ctx context.Context, email, password string) (SuperadminResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !validEmail(email) {
		return "", errors.New("email is not valid")
	}
	if utf8.RuneCountInString(password) < minSuperadminPasswordLen {
		return "", fmt.Errorf("password must be at least %d characters", minSuperadminPasswordLen)
	}
	if len(password) > maxPasswordBytes {
		return "", fmt.Errorf("password must be at most %d bytes", maxPasswordBytes)
	}

	existing, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return "", fmt.Errorf("look up superadmin: %w", err)
	}
	if err == nil && existing.Role != models.RoleSuperadmin {
		return SuperadminConflict, nil
	}

	hash, err := HashPassword(password)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	if existing != nil {
		if err := s.repo.SetPassword(ctx, existing.ID, hash, false); err != nil {
			return "", fmt.Errorf("set superadmin password: %w", err)
		}
		return SuperadminPasswordReset, nil
	}
	u := &models.User{
		FullName:     "Оператор платформы",
		Email:        email,
		Phone:        superadminPhone(email),
		PasswordHash: hash,
	}
	if err := s.repo.CreateSuperadmin(ctx, u); err != nil {
		return "", fmt.Errorf("create superadmin: %w", err)
	}
	return SuperadminCreated, nil
}

// superadminPhone is a unique placeholder: users.phone is NOT NULL UNIQUE, and
// the operator logs in by email. It can't collide with a real phone number.
func superadminPhone(email string) string {
	p := "superadmin:" + email
	if len(p) > maxPhoneLen {
		return "superadmin:" + uuid.NewString()
	}
	return p
}

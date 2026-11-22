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

// SuperadminResult says what EnsureSuperadmin did.
type SuperadminResult string

const (
	SuperadminCreated SuperadminResult = "created"
	SuperadminExists  SuperadminResult = "exists"
	// SuperadminConflict: the email belongs to a user with another role; nothing was changed.
	SuperadminConflict SuperadminResult = "conflict"
)

// EnsureSuperadmin makes sure a platform operator with this email exists
// (startup bootstrap from SUPERADMIN_EMAIL / SUPERADMIN_PASSWORD). An existing
// superadmin keeps their current password; a user with another role is never
// promoted. The password is never logged.
func (s *Service) EnsureSuperadmin(ctx context.Context, email, password string) (SuperadminResult, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if !validEmail(email) {
		return "", errors.New("SUPERADMIN_EMAIL is not a valid email")
	}
	if utf8.RuneCountInString(password) < minSuperadminPasswordLen {
		return "", fmt.Errorf("SUPERADMIN_PASSWORD must be at least %d characters", minSuperadminPasswordLen)
	}

	existing, err := s.repo.GetUserByEmail(ctx, email)
	switch {
	case err == nil && existing.Role == models.RoleSuperadmin:
		return SuperadminExists, nil
	case err == nil:
		return SuperadminConflict, nil
	case !errors.Is(err, ErrNotFound):
		return "", fmt.Errorf("look up superadmin: %w", err)
	}

	hash, err := HashPassword(password)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
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

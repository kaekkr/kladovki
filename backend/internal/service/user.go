package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/kaekkr/kladovki/internal/models"
)

type CreateUserInput struct {
	FullName string  `json:"full_name"`
	Phone    string  `json:"phone"`
	Email    string  `json:"email"`
	Password string  `json:"password"`
	JKID     *string `json:"jk_id,omitempty"`
}

func (s *Service) CreateUser(ctx context.Context, in CreateUserInput) (*models.User, error) {
	fullName := strings.TrimSpace(in.FullName)
	phone := strings.TrimSpace(in.Phone)
	email := strings.TrimSpace(in.Email)
	password := strings.TrimSpace(in.Password)

	if phone == "" || password == "" {
		return nil, ErrInvalid
	}

	hash, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("service.CreateUser hash password: %w", err)
	}

	user := &models.User{
		FullName:     fullName,
		Phone:        phone,
		Email:        email,
		PasswordHash: hash,
		Role:         models.RoleResident,
		JKID:         in.JKID,
	}

	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("service.CreateUser: %w", err)
	}

	return user, nil
}

func (s *Service) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrInvalid
	}

	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// MaxLoginLen: a login identifier (ИИН, phone or email) longer than this is a 400.
const MaxLoginLen = 255

// ValidLoginIdentifier: at most MaxLoginLen characters, valid UTF-8, no NUL.
// Checked before the limiter or the DB see the identifier.
func ValidLoginIdentifier(login string) bool {
	return utf8.ValidString(login) && !strings.ContainsRune(login, 0) &&
		utf8.RuneCountInString(login) <= MaxLoginLen
}

// LoginKey is the identifier in one canonical spelling, so the failed-login
// counter is shared by "87001234567" and "+7 700 123 45 67". It is hashed:
// the limiter keeps fixed-size keys and no identifiers in memory.
func LoginKey(login string) string {
	return "login:" + hashKey(canonicalLogin(login))
}

func canonicalLogin(login string) string {
	login = strings.TrimSpace(login)
	if ValidIIN(login) {
		return "iin:" + login
	}
	if strings.Contains(login, "@") {
		return "email:" + strings.ToLower(login)
	}
	if p, ok := NormalizePhone(login); ok {
		return "phone:" + p
	}
	return "raw:" + login
}

func hashKey(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// GetUserByLogin looks the identifier up by its type only, so one user's
// phone can never shadow another user's ИИН or email:
// 12 digits → ИИН; contains "@" → email; anything else → phone.
func (s *Service) GetUserByLogin(ctx context.Context, login string) (*models.User, error) {
	login = strings.TrimSpace(login)
	switch {
	case login == "":
		return nil, ErrInvalid
	case ValidIIN(login):
		return s.repo.GetUserByIIN(ctx, login)
	case strings.Contains(login, "@"):
		return s.repo.GetUserByEmail(ctx, login)
	}
	canonical, _ := NormalizePhone(login)
	return s.repo.GetUserByPhoneLogin(ctx, canonical, login)
}

// SessionUser re-reads what authorization needs for one request.
func (s *Service) SessionUser(ctx context.Context, userID string) (*models.SessionUser, error) {
	return s.repo.GetSessionUser(ctx, userID)
}

// Logout ends every session of the user that was issued with tokenVersion.
func (s *Service) Logout(ctx context.Context, userID string, tokenVersion int) error {
	return s.repo.RevokeSessions(ctx, userID, tokenVersion)
}

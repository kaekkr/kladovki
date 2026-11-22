package service

import (
	"context"
	"errors"
	"strings"

	"github.com/kaekkr/kladovki/internal/repository"
)

// ErrInvalidPhone: registration and ОСИ accounts accept only Kazakhstan numbers.
var ErrInvalidPhone = errors.New("Укажите номер телефона в формате +7XXXXXXXXXX.")

// NormalizePhone turns a Kazakhstan number typed as +7XXXXXXXXXX, 8XXXXXXXXXX
// or 7XXXXXXXXXX (spaces, dashes and parentheses allowed) into the canonical
// +7XXXXXXXXXX. ok is false for anything else — notably an ИИН or an email,
// so a phone can never collide with another user's login identifier.
func NormalizePhone(raw string) (phone string, ok bool) {
	var b strings.Builder
	for i, ch := range strings.TrimSpace(raw) {
		switch {
		case ch >= '0' && ch <= '9':
			b.WriteRune(ch)
		case ch == '+' && i == 0:
			b.WriteRune(ch)
		case ch == ' ' || ch == '-' || ch == '(' || ch == ')':
		default:
			return "", false
		}
	}
	s := b.String()
	switch {
	case len(s) == 12 && strings.HasPrefix(s, "+7"):
		return s, true
	case len(s) == 11 && (s[0] == '8' || s[0] == '7'):
		return "+7" + s[1:], true
	}
	return "", false
}

// UpdatePhone changes the user's own phone (Kaspi invoices go to it). The
// current password is checked first: ErrWrongPassword when it doesn't match.
func (s *Service) UpdatePhone(ctx context.Context, userID, currentPassword, raw string) (string, error) {
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return "", err
	}
	if !CheckPassword(u.PasswordHash, currentPassword) {
		return "", ErrWrongPassword
	}
	phone, ok := NormalizePhone(raw)
	if !ok {
		return "", ErrInvalidPhone
	}
	switch err := s.repo.UpdatePhone(ctx, userID, phone); {
	case err == nil:
		return phone, nil
	case errors.Is(err, repository.ErrPhoneTaken):
		return "", ErrPhoneTaken
	case errors.Is(err, repository.ErrNotFound):
		return "", ErrNotFound
	default:
		return "", err
	}
}

package service

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
)

var ErrWrongPassword = errors.New("Текущий пароль указан неверно.")

// ChangePassword: any user changes their own password (current one required).
func (s *Service) ChangePassword(ctx context.Context, userID, current, next string) error {
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if !CheckPassword(u.PasswordHash, current) {
		return ErrWrongPassword
	}
	if len([]rune(next)) < minPasswordLen {
		return ErrWeakPassword
	}
	hash, err := HashPassword(next)
	if err != nil {
		return err
	}
	return s.repo.SetPassword(ctx, userID, hash, false)
}

// ResetResidentPassword: the ОСИ sets a temporary password for a resident of
// their ЖК (the login page tells residents to ask the ОСИ). It is returned
// once, to hand over in person; it is never sent through notification
// channels. The resident must choose a new password after logging in.
func (s *Service) ResetResidentPassword(ctx context.Context, jkID, userID string) (string, error) {
	temp, err := temporaryPassword(10)
	if err != nil {
		return "", err
	}
	hash, err := HashPassword(temp)
	if err != nil {
		return "", err
	}
	if err := s.repo.ResetResidentPassword(ctx, userID, jkID, hash); err != nil {
		return "", mapRepoErr(err)
	}
	return temp, nil
}

// temporaryPassword avoids look-alike characters (0/O, 1/l/I) — it is read aloud or written down.
func temporaryPassword(n int) (string, error) {
	const alphabet = "abcdefghjkmnpqrstuvwxyz23456789"
	b := make([]byte, n)
	for i := range b {
		k, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		b[i] = alphabet[k.Int64()]
	}
	return string(b), nil
}

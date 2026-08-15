package service

import (
	"errors"

	"github.com/kaekkr/kladovki/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrForbidden     = errors.New("forbidden")
	ErrInvalid       = errors.New("invalid")
	ErrLocked        = errors.New("storage locked or occupied")
)

type Service struct {
	repo *repository.Repo
}

func New(repo *repository.Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) Repo() *repository.Repo { return s.repo }

// ---------- Auth Helpers ----------

func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

package service

import (
	"errors"

	"github.com/kaekkr/kladovki/internal/notify"
	"github.com/kaekkr/kladovki/internal/repository"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrAlreadyExists      = errors.New("already exists")
	ErrForbidden          = errors.New("forbidden")
	ErrInvalid            = errors.New("invalid")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrLocked             = errors.New("storage locked or occupied")
	ErrTransferPending    = errors.New("storage transfer pending")
	ErrPaymentPending     = errors.New("payment already pending")
)

type Service struct {
	repo      *repository.Repo
	notifyCfg notify.Config
	channels  map[string]notify.Channel
}

func New(repo *repository.Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) Repo() *repository.Repo { return s.repo }

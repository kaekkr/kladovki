package service

import (
	"errors"

	"github.com/kaekkr/kladovki/internal/notify"
	"github.com/kaekkr/kladovki/internal/repository"
)

var (
	// Same value as the repository's, so handlers recognise "not found" from either layer.
	ErrNotFound           = repository.ErrNotFound
	ErrAlreadyExists      = errors.New("already exists")
	ErrForbidden          = errors.New("forbidden")
	ErrInvalid            = errors.New("invalid")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrLocked             = errors.New("storage locked or occupied")
	ErrTransferPending    = errors.New("storage transfer pending")
	ErrPaymentPending     = errors.New("payment already pending")
	ErrStorageInUse       = errors.New("Кладовку с историей аренд удалить нельзя — она нужна для отчётов и споров.")
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

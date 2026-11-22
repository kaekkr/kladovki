package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/kaekkr/kladovki/internal/models"
	"github.com/kaekkr/kladovki/internal/repository"
)

// Platform operator (superadmin) use cases: ЖК and their ОСИ accounts.
// Temporary passwords are returned once, to hand over in person; they are
// never logged or sent through notification channels.

// NewAccount is a created ОСИ account and its one-time temporary password.
type NewAccount struct {
	User              *models.User
	TemporaryPassword string
}

func (s *Service) ListJKSummaries(ctx context.Context) ([]*models.JKSummary, error) {
	return s.repo.ListJKSummaries(ctx)
}

func (s *Service) GetJKSummary(ctx context.Context, id string) (*models.JKSummary, error) {
	return s.repo.GetJKSummary(ctx, id)
}

// CreateJKWithAdmin creates a ЖК and its first ОСИ in one transaction.
// No tariff row is created: the ОСИ sets the price in Настройки before
// residents can book (booking refuses a ЖК without a tariff).
func (s *Service) CreateJKWithAdmin(ctx context.Context, details JKDetails, admin AccountInput) (*models.JK, *NewAccount, error) {
	d, err := normalizeJKDetails(details)
	if err != nil {
		return nil, nil, err
	}
	user, temp, err := newAdminUser(admin)
	if err != nil {
		return nil, nil, err
	}
	jk := &models.JK{Name: d.Name, Address: d.Address, CadastralNumber: d.CadastralNumber}
	if err := s.repo.CreateJKWithAdmin(ctx, jk, user); err != nil {
		return nil, nil, mapPlatformErr(err)
	}
	return jk, &NewAccount{User: user, TemporaryPassword: temp}, nil
}

// UpdateJKDetails edits name, address and cadastral number of a ЖК.
func (s *Service) UpdateJKDetails(ctx context.Context, id string, details JKDetails) (*models.JKSummary, error) {
	d, err := normalizeJKDetails(details)
	if err != nil {
		return nil, err
	}
	jk := &models.JK{ID: id, Name: d.Name, Address: d.Address, CadastralNumber: d.CadastralNumber}
	if err := s.repo.UpdateJKDetails(ctx, jk); err != nil {
		return nil, mapPlatformErr(err)
	}
	return s.repo.GetJKSummary(ctx, id)
}

// ListJKAdmins lists the ОСИ accounts of a ЖК (404 when the ЖК doesn't exist).
func (s *Service) ListJKAdmins(ctx context.Context, jkID string) ([]*models.User, error) {
	if _, err := s.repo.GetJKByID(ctx, jkID); err != nil {
		return nil, err
	}
	return s.repo.ListJKAdmins(ctx, jkID)
}

// AddJKAdmin creates another ОСИ account for an existing ЖК.
func (s *Service) AddJKAdmin(ctx context.Context, jkID string, admin AccountInput) (*NewAccount, error) {
	user, temp, err := newAdminUser(admin)
	if err != nil {
		return nil, err
	}
	if err := s.repo.AddJKAdmin(ctx, jkID, user); err != nil {
		return nil, mapPlatformErr(err)
	}
	return &NewAccount{User: user, TemporaryPassword: temp}, nil
}

// ResetAdminPassword gives an ОСИ account a new temporary password.
func (s *Service) ResetAdminPassword(ctx context.Context, userID string) (string, error) {
	temp, hash, err := newTemporaryPassword()
	if err != nil {
		return "", err
	}
	if err := s.repo.ResetAdminPassword(ctx, userID, hash); err != nil {
		return "", mapPlatformErr(err)
	}
	return temp, nil
}

// newAdminUser validates the input and builds an ОСИ user with a temporary password.
func newAdminUser(in AccountInput) (*models.User, string, error) {
	acc, err := normalizeAccount(in)
	if err != nil {
		return nil, "", err
	}
	temp, hash, err := newTemporaryPassword()
	if err != nil {
		return nil, "", err
	}
	return &models.User{
		FullName:           acc.FullName,
		Email:              acc.Email,
		Phone:              acc.Phone,
		PasswordHash:       hash,
		MustChangePassword: true,
	}, temp, nil
}

func newTemporaryPassword() (temp, hash string, err error) {
	temp, err = temporaryPassword(tempPasswordLen)
	if err != nil {
		return "", "", fmt.Errorf("temporary password: %w", err)
	}
	hash, err = HashPassword(temp)
	if err != nil {
		return "", "", fmt.Errorf("hash password: %w", err)
	}
	return temp, hash, nil
}

func mapPlatformErr(err error) error {
	switch {
	case errors.Is(err, repository.ErrEmailTaken):
		return ErrEmailTaken
	case errors.Is(err, repository.ErrPhoneTaken):
		return ErrPhoneTaken
	case errors.Is(err, repository.ErrCadastralTaken):
		return ErrCadastralTaken
	}
	return mapRepoErr(err)
}

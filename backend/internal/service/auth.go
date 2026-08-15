package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kaekkr/kladovki/internal/models"
)

func (s *Service) RegisterResident(ctx context.Context, fullName, phone, email, iin, password, jkID string) (*models.User, error) {
	if _, err := s.repo.GetUserByEmail(ctx, email); err == nil {
		return nil, fmt.Errorf("%w: email already in use", ErrAlreadyExists)
	}
	if _, err := s.repo.GetUserByPhone(ctx, phone); err == nil {
		return nil, fmt.Errorf("%w: phone number already in use", ErrAlreadyExists)
	}
	if _, err := s.repo.GetUserByIIN(ctx, iin); err == nil {
		return nil, fmt.Errorf("%w: IIN already registered", ErrAlreadyExists)
	}

	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	var targetJKID *string
	if jkID != "" {
		targetJKID = &jkID
	}

	u := &models.User{
		ID:           uuid.New().String(),
		Roles:        []models.Role{models.RoleResident},
		FullName:     fullName,
		Phone:        phone,
		Email:        email,
		IIN:          &iin,
		JKID:         targetJKID,
		PasswordHash: hash,
		CreatedAt:    time.Now(),
	}

	if err := s.repo.CreateUser(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Service) Login(ctx context.Context, login, password string) (*models.User, error) {
	u, err := s.repo.GetUserByEmail(ctx, login)
	if err != nil {
		u, err = s.repo.GetUserByPhone(ctx, login)
		if err != nil {
			return nil, ErrNotFound
		}
	}

	if !CheckPassword(u.PasswordHash, password) {
		return nil, ErrForbidden
	}
	return u, nil
}

func (s *Service) EgovStub(ctx context.Context, iin string) ([]models.JK, error) {
	var targetJKName string
	switch iin {
	case "123456789012":
		targetJKName = "ЖК Сыганак"
	default:
		return []models.JK{}, nil
	}

	jk, err := s.repo.GetJKByName(ctx, targetJKName)
	if err != nil {
		return []models.JK{}, nil
	}

	return []models.JK{*jk}, nil
}

func (s *Service) ListJKs(ctx context.Context) ([]*models.JK, error) {
	return s.repo.ListJKs(ctx)
}

func (s *Service) UpdateUserJKID(ctx context.Context, userID, jkID string) error {
	return s.repo.UpdateUserJKID(ctx, userID, jkID)
}

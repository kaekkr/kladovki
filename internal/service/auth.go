package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kaekkr/kladovki/internal/models"
)

func (s *Service) RegisterResident(ctx context.Context, fullName, phone, email, iin, password string) (*models.User, error) {
	if _, err := s.repo.GetUserByEmail(ctx, email); err == nil {
		return nil, fmt.Errorf("%w: email", ErrAlreadyExists)
	}

	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	u := &models.User{
		ID:           uuid.New().String(), // <-- Primary Key fix for Postgres
		Role:         models.RoleResident,
		FullName:     fullName,
		Phone:        phone,
		Email:        email,
		IIN:          &iin,
		PasswordHash: hash,
		CreatedAt:    time.Now(),
	}

	// Fetch stubbed JK and link it to the new user
	jks, err := s.EgovStub(ctx, iin)
	if err == nil && len(jks) > 0 {
		u.JKID = &jks[0].ID
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
	// Try fetching by exact name first
	jk, err := s.repo.GetJKByName(ctx, "ЖК Сыганак")
	if err == nil {
		return []models.JK{*jk}, nil
	}

	// Fallback to fetching the first available JK in the DB (e.g. 'jk-1')
	jk, err = s.repo.GetJKByID(ctx, "jk-1")
	if err == nil {
		return []models.JK{*jk}, nil
	}

	return []models.JK{}, nil
}

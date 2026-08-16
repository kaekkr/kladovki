package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kaekkr/kladovki/internal/models"
)

type CreateUserInput struct {
	FullName string
	Phone    string
	Email    string
	Password string
	Role     models.Role
	JKID     *string
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
		Role:         in.Role,
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

func (s *Service) GetUserByLogin(ctx context.Context, login string) (*models.User, error) {
	login = strings.TrimSpace(login)
	if login == "" {
		return nil, ErrInvalid
	}

	user, err := s.repo.GetUserByLogin(ctx, login)
	if err != nil {
		return nil, err
	}

	return user, nil
}

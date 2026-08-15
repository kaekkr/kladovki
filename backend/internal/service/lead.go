package service

import (
	"context"
	"strings"

	"github.com/kaekkr/kladovki/internal/models"
)

type CreateLeadInput struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
}

func (s *Service) CreateLead(ctx context.Context, in CreateLeadInput) (*models.Lead, error) {
	fullName := strings.TrimSpace(in.FullName)
	phone := strings.TrimSpace(in.Phone)
	email := strings.TrimSpace(in.Email)

	if fullName == "" || phone == "" {
		return nil, ErrInvalid
	}

	lead := &models.Lead{
		FullName: fullName,
		Phone:    phone,
		Email:    email,
	}

	if err := s.repo.CreateLead(ctx, lead); err != nil {
		return nil, err
	}

	return lead, nil
}

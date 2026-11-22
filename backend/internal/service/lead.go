package service

import (
	"context"
	"strings"
	"unicode/utf8"

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
	// Column sizes of the leads table: answer 400, not a DB error.
	switch {
	case !cleanText(fullName, phone, email):
		return nil, ErrInvalidText
	case utf8.RuneCountInString(fullName) > maxNameLen:
		return nil, invalid("Имя слишком длинное.")
	case utf8.RuneCountInString(phone) > maxPhoneLen:
		return nil, invalid("Номер телефона слишком длинный.")
	case utf8.RuneCountInString(email) > maxEmailLen:
		return nil, invalid("Email слишком длинный.")
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

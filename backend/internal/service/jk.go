package service

import (
	"context"
	"strings"

	"github.com/kaekkr/kladovki/internal/models"
)

func (s *Service) GetJKByID(ctx context.Context, id string) (*models.JK, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrInvalid
	}

	return s.repo.GetJKByID(ctx, id)
}

func (s *Service) GetJKByName(ctx context.Context, name string) (*models.JK, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalid
	}

	return s.repo.GetJKByName(ctx, name)
}

func (s *Service) ListJKs(ctx context.Context) ([]*models.JK, error) {
	return s.repo.ListJKs(ctx)
}

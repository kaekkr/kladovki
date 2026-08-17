package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kaekkr/kladovki/internal/models"
)

type SetTariffInput struct {
	JKID   string `json:"jk_id"`
	Amount int64  `json:"amount"`
}

func (s *Service) SetTariff(ctx context.Context, in SetTariffInput) error {
	jkID := strings.TrimSpace(in.JKID)
	if jkID == "" || in.Amount <= 0 {
		return ErrInvalid
	}

	if err := s.repo.SetTariff(ctx, jkID, in.Amount); err != nil {
		return fmt.Errorf("service.SetTariff: %w", err)
	}

	return nil
}

func (s *Service) GetTariff(ctx context.Context, jkID string) (int64, error) {
	jkID = strings.TrimSpace(jkID)
	if jkID == "" {
		return 0, ErrInvalid
	}

	return s.repo.GetTariff(ctx, jkID)
}

func (s *Service) GetTariffModel(ctx context.Context, jkID string) (*models.Tariff, error) {
	jkID = strings.TrimSpace(jkID)
	if jkID == "" {
		return nil, ErrInvalid
	}

	return s.repo.GetTariffModel(ctx, jkID)
}

func (s *Service) TariffHistory(ctx context.Context, jkID string) ([]models.TariffHistory, error) {
	jkID = strings.TrimSpace(jkID)
	if jkID == "" {
		return nil, ErrInvalid
	}

	return s.repo.TariffHistory(ctx, jkID)
}

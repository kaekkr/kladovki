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

func (s *Service) GetTariffPreview(ctx context.Context, jkID string, newAmount int64) (models.TariffPreviewResponse, error) {
	jkID = strings.TrimSpace(jkID)
	if jkID == "" || newAmount <= 0 {
		return models.TariffPreviewResponse{}, ErrInvalid
	}
	return s.repo.TariffDecreasePreview(ctx, jkID, newAmount)
}

// SetTariff saves the new tariff. Already-paid periods keep their price; when
// the tariff goes down every affected tenant gets a request to choose between
// extending the period and a refund. Returns how many tenants were asked.
func (s *Service) SetTariff(ctx context.Context, in SetTariffInput) (int64, error) {
	jkID := strings.TrimSpace(in.JKID)
	if jkID == "" || in.Amount <= 0 {
		return 0, ErrInvalid
	}

	oldTariff, _ := s.repo.GetTariff(ctx, jkID)

	if err := s.repo.SetTariff(ctx, jkID, in.Amount); err != nil {
		return 0, fmt.Errorf("service.SetTariff: %w", err)
	}

	n, err := s.repo.CreateTariffRecalcRequests(ctx, jkID, in.Amount)
	if err != nil {
		return 0, fmt.Errorf("service.SetTariff recalc requests: %w", err)
	}
	s.notifyTariffChange(ctx, jkID, oldTariff, in.Amount)
	return n, nil
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

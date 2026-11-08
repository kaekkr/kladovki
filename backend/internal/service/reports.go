package service

import (
	"context"
	"time"

	"github.com/kaekkr/kladovki/internal/models"
	"github.com/kaekkr/kladovki/internal/repository"
)

func (s *Service) IncomeReport(ctx context.Context, jkID string, from, to time.Time) ([]repository.IncomeRow, error) {
	return s.repo.IncomeRows(ctx, jkID, from, to)
}

func (s *Service) StorageReport(ctx context.Context, jkID string) ([]repository.StorageRow, error) {
	return s.repo.StorageRows(ctx, jkID)
}

func (s *Service) ListPaymentsByJKPeriod(ctx context.Context, jkID string, from, to *time.Time) ([]*models.PaymentResponse, error) {
	if jkID == "" {
		return nil, ErrInvalid
	}
	return s.repo.ListPaymentsByJK(ctx, jkID, from, to)
}

package service

import (
	"context"

	"github.com/kaekkr/kladovki/internal/models"
)

func (s *Service) ListPaymentsByJK(ctx context.Context, jkID string) ([]*models.PaymentResponse, error) {
	if jkID == "" {
		return nil, ErrInvalid
	}
	return s.repo.ListPaymentsByJK(ctx, jkID)
}

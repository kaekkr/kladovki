package service

import (
	"context"

	"github.com/kaekkr/kladovki/internal/models"
)

func (s *Service) GetStorageByID(ctx context.Context, id string) (*models.Storage, error) {
	return s.repo.GetStorageByID(ctx, id)
}

func (s *Service) GetTariff(ctx context.Context, jkID string) (int64, error) {
	return s.repo.GetTariff(ctx, jkID)
}

func (s *Service) SetTariff(ctx context.Context, jkID string, amount int64) error {
	return s.repo.SetTariff(ctx, jkID, amount)
}

func (s *Service) UpdateStorage(ctx context.Context, storage *models.Storage) error {
	// 1. Fetch current record to prevent wiping unsubmitted fields (like JKID or Price)
	existing, err := s.repo.GetStorageByID(ctx, storage.ID)
	if err != nil {
		return err
	}

	// 2. Keep immutable fields intact
	storage.JKID = existing.JKID
	storage.Price = existing.Price
	if storage.CreatedAt.IsZero() {
		storage.CreatedAt = existing.CreatedAt
	}

	return s.repo.UpdateStorage(ctx, storage)
}

func (s *Service) DeleteStorage(ctx context.Context, id string) error {
	return s.repo.DeleteStorage(ctx, id)
}

package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kaekkr/kladovki/internal/models"
)

type CreateStorageInput struct {
	JKID     string  `json:"jk_id"`
	Number   string  `json:"number"`
	Area     float64 `json:"area"`
	Floor    int     `json:"floor"`
	Entrance int     `json:"entrance"`
}

type UpdateStorageInput struct {
	ID       string               `json:"id"`
	Number   string               `json:"number"`
	Area     float64              `json:"area"`
	Floor    int                  `json:"floor"`
	Entrance int                  `json:"entrance"`
	Status   models.StorageStatus `json:"status"`
}

func (s *Service) CreateStorage(ctx context.Context, in CreateStorageInput) (*models.Storage, error) {
	jkID := strings.TrimSpace(in.JKID)
	number := strings.TrimSpace(in.Number)

	if jkID == "" || number == "" || in.Area <= 0 {
		return nil, ErrInvalid
	}

	storage := &models.Storage{
		JKID:     jkID,
		Number:   number,
		Area:     in.Area,
		Floor:    in.Floor,
		Entrance: in.Entrance,
		Status:   models.StatusFree,
	}

	if err := s.repo.CreateStorage(ctx, storage); err != nil {
		return nil, fmt.Errorf("service.CreateStorage: %w", err)
	}

	return storage, nil
}

func (s *Service) GetStorageByID(ctx context.Context, id string) (*models.Storage, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrInvalid
	}

	storage, err := s.repo.GetStorageByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return storage, nil
}

func (s *Service) ListStoragesByJK(ctx context.Context, jkID string) ([]*models.Storage, error) {
	jkID = strings.TrimSpace(jkID)
	if jkID == "" {
		return nil, ErrInvalid
	}

	return s.repo.ListStoragesByJK(ctx, jkID)
}

func (s *Service) GetAvailableStoragesByJK(ctx context.Context, jkID string) ([]*models.Storage, error) {
	jkID = strings.TrimSpace(jkID)
	if jkID == "" {
		return nil, ErrInvalid
	}

	return s.repo.GetAvailableStoragesByJK(ctx, jkID)
}

func (s *Service) UpdateStorage(ctx context.Context, in UpdateStorageInput) (*models.Storage, error) {
	id := strings.TrimSpace(in.ID)
	number := strings.TrimSpace(in.Number)

	if id == "" || number == "" || in.Area <= 0 {
		return nil, ErrInvalid
	}

	// 1. Fetch current record to preserve immutable fields like JKID
	existing, err := s.repo.GetStorageByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 2. Prepare storage model with updated values
	storage := &models.Storage{
		ID:        existing.ID,
		JKID:      existing.JKID,
		Number:    number,
		Area:      in.Area,
		Floor:     in.Floor,
		Entrance:  in.Entrance,
		Status:    in.Status,
		CreatedAt: existing.CreatedAt,
	}

	if storage.Status == "" {
		storage.Status = existing.Status
	}

	if err := s.repo.UpdateStorage(ctx, storage); err != nil {
		return nil, fmt.Errorf("service.UpdateStorage: %w", err)
	}

	return storage, nil
}

func (s *Service) DeleteStorage(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrInvalid
	}

	if err := s.repo.DeleteStorage(ctx, id); err != nil {
		return fmt.Errorf("service.DeleteStorage: %w", err)
	}

	return nil
}

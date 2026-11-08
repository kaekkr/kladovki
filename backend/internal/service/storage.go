package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kaekkr/kladovki/internal/models"
	"github.com/kaekkr/kladovki/internal/notify"
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

	if taken, err := s.repo.StorageNumberTaken(ctx, jkID, number, ""); err != nil {
		return nil, fmt.Errorf("service.CreateStorage check number: %w", err)
	} else if taken {
		return nil, ErrAlreadyExists
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

	if taken, err := s.repo.StorageNumberTaken(ctx, existing.JKID, number, existing.ID); err != nil {
		return nil, fmt.Errorf("service.UpdateStorage check number: %w", err)
	} else if taken {
		return nil, ErrAlreadyExists
	}

	// Agreement 4.2: while occupied or being booked, parameters are frozen
	// until the rental ends.
	if existing.Status != models.StatusFree &&
		(number != existing.Number || in.Area != existing.Area ||
			in.Floor != existing.Floor || in.Entrance != existing.Entrance) {
		return nil, ErrLocked
	}

	// 2. Prepare storage model with updated values
	storage := &models.Storage{
		ID:       existing.ID,
		JKID:     existing.JKID,
		Number:   number,
		Area:     in.Area,
		Floor:    in.Floor,
		Entrance: in.Entrance,
		// Status is derived from rentals (free/locked/occupied), never set by hand.
		Status:    existing.Status,
		CreatedAt: existing.CreatedAt,

		TransferToIIN: existing.TransferToIIN,
		TransferUntil: existing.TransferUntil,
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

func (s *Service) SetStorageTransfer(
	ctx context.Context,
	storageID string,
	iin string,
) error {
	storageID = strings.TrimSpace(storageID)
	iin = strings.TrimSpace(iin)

	if storageID == "" || len(iin) != 12 {
		return ErrInvalid
	}

	for _, ch := range iin {
		if ch < '0' || ch > '9' {
			return ErrInvalid
		}
	}

	storage, err := s.repo.GetStorageByID(ctx, storageID)
	if err != nil {
		return err
	}

	if storage.Status != models.StatusOccupied {
		return ErrInvalid
	}

	rental, err := s.repo.GetActiveRentalByStorage(ctx, storageID)
	if err != nil {
		return err
	}

	if time.Now().Before(rental.EndsAt) {
		return ErrInvalid
	}

	if err := s.repo.SetStorageTransfer(ctx, storageID, iin); err != nil {
		return fmt.Errorf("service.SetStorageTransfer: %w", err)
	}
	// Agreement п. 6.2: passing a storage to someone else must not happen behind the ОСИ.
	s.notifyJKAdmins(ctx, storage.JKID, KindTransferNew, notify.Message{
		Title: fmt.Sprintf("Передача кладовки №%s", storage.Number),
		Body: fmt.Sprintf("%s указал(а) нового владельца кладовки №%s (ИИН %s). Новый владелец сможет оформить аренду до %s.",
			rental.UserName, storage.Number, iin, kzDate(rental.EndsAt.AddDate(0, 0, 14))),
		Link: "/admin/storages",
	}, fmt.Sprintf("transfer:%s:%s", storageID, iin))

	return nil
}

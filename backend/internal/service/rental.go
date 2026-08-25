package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/kaekkr/kladovki/internal/models"
)

const LockDuration = 2 * time.Minute

type LockStorageInput struct {
	StorageID string `json:"storage_id"`
	UserID    string `json:"user_id"`
	Months    int    `json:"months"`
}

type ConfirmPaymentInput struct {
	RentalID string `json:"rental_id"`
	UserID   string `json:"user_id"`
}

func (s *Service) LockStorage(
	ctx context.Context,
	in LockStorageInput,
) (*models.Rental, error) {
	storageID := strings.TrimSpace(in.StorageID)
	userID := strings.TrimSpace(in.UserID)

	if storageID == "" || userID == "" {
		return nil, ErrInvalid
	}

	months := max(in.Months, 1)

	st, err := s.repo.GetStorageByID(ctx, storageID)
	if err != nil {
		return nil, err
	}

	tariffAmount, err := s.repo.GetTariff(ctx, st.JKID)
	if err != nil {
		return nil, fmt.Errorf(
			"service.LockStorage get tariff: %w",
			err,
		)
	}

	if tariffAmount == 0 {
		return nil, fmt.Errorf(
			"service.LockStorage: tariff not set for jk_id %s",
			st.JKID,
		)
	}

	pricePerMonth := int64(
		math.Round(st.Area * float64(tariffAmount)),
	)

	return s.repo.LockStorage(
		ctx,
		storageID,
		userID,
		st.JKID,
		months,
		pricePerMonth,
		LockDuration,
	)
}

func (s *Service) ConfirmPayment(
	ctx context.Context,
	in ConfirmPaymentInput,
) (*models.Rental, error) {
	rentalID := strings.TrimSpace(in.RentalID)
	userID := strings.TrimSpace(in.UserID)

	if rentalID == "" || userID == "" {
		return nil, ErrInvalid
	}

	return s.repo.ConfirmRentalPayment(
		ctx,
		rentalID,
		userID,
	)
}

func (s *Service) ForceReleaseLocked(ctx context.Context, rentalID, jkID string) (*models.Rental, error) {
	rentalID = strings.TrimSpace(rentalID)
	jkID = strings.TrimSpace(jkID)
	if rentalID == "" || jkID == "" {
		return nil, ErrInvalid
	}
	return s.repo.ForceReleaseLocked(ctx, rentalID, jkID)
}

func (s *Service) GetRentalByID(ctx context.Context, id string) (*models.Rental, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrInvalid
	}

	return s.repo.GetRentalByID(ctx, id)
}

func (s *Service) GetActiveRentalByStorage(ctx context.Context, storageID string) (*models.Rental, error) {
	storageID = strings.TrimSpace(storageID)
	if storageID == "" {
		return nil, ErrInvalid
	}

	return s.repo.GetActiveRentalByStorage(ctx, storageID)
}

func (s *Service) ListRentalsByJK(ctx context.Context, jkID string) ([]*models.Rental, error) {
	jkID = strings.TrimSpace(jkID)

	if jkID == "" {
		return nil, ErrInvalid
	}

	return s.repo.ListRentalsByJK(ctx, jkID)
}

func (s *Service) ListActiveRentalsByJK(ctx context.Context, jkID string) ([]*models.Rental, error) {
	jkID = strings.TrimSpace(jkID)
	if jkID == "" {
		return nil, ErrInvalid
	}

	return s.repo.ListActiveRentalsByJK(ctx, jkID)
}

func (s *Service) CancelRental(
	ctx context.Context,
	rentalID string,
	jkID string,
) (*models.Rental, error) {
	rentalID = strings.TrimSpace(rentalID)
	jkID = strings.TrimSpace(jkID)

	if rentalID == "" || jkID == "" {
		return nil, ErrInvalid
	}

	return s.repo.CancelRental(
		ctx,
		rentalID,
		jkID,
	)
}

func (s *Service) ExpireRentals(ctx context.Context) error {
	return s.repo.ExpireRentals(ctx)
}

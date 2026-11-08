package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/kaekkr/kladovki/internal/models"
)

// LockDuration is the default hold before payment; each ЖК can override it.
const LockDuration = 2 * time.Minute

// Bounds for the ОСИ "lock duration" setting, in minutes.
const (
	MinLockMinutes = 1
	MaxLockMinutes = 60
)

func (s *Service) lockDuration(ctx context.Context, jkID string) time.Duration {
	if n, err := s.repo.LockMinutes(ctx, jkID); err == nil && n > 0 {
		return time.Duration(n) * time.Minute
	}
	return LockDuration
}

func (s *Service) GetRentalSettings(ctx context.Context, jkID string) (models.RentalSettings, error) {
	n, err := s.repo.LockMinutes(ctx, jkID)
	if err != nil {
		return models.RentalSettings{}, err
	}
	return models.RentalSettings{LockDurationMinutes: n}, nil
}

func (s *Service) SetRentalSettings(ctx context.Context, jkID string, in models.RentalSettings) error {
	if in.LockDurationMinutes < MinLockMinutes || in.LockDurationMinutes > MaxLockMinutes {
		return ErrInvalid
	}
	return s.repo.SetLockMinutes(ctx, jkID, in.LockDurationMinutes)
}

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

	if in.Months < 1 || in.Months > 120 {
		return nil, ErrInvalid
	}

	months := in.Months

	st, err := s.repo.GetStorageByID(ctx, storageID)
	if err != nil {
		return nil, err
	}

	if st.TransferToIIN != "" {
		if st.TransferUntil == nil || time.Now().After(*st.TransferUntil) {
			return nil, ErrInvalid
		}

		user, err := s.repo.GetUserByID(ctx, userID)
		if err != nil {
			return nil, err
		}

		if user.IIN != st.TransferToIIN {
			return nil, ErrInvalid
		}
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
		s.lockDuration(ctx, st.JKID),
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

func (s *Service) ReleaseOverdueRental(
	ctx context.Context,
	rentalID string,
	jkID string,
) (*models.Rental, error) {
	rentalID = strings.TrimSpace(rentalID)
	jkID = strings.TrimSpace(jkID)

	if rentalID == "" || jkID == "" {
		return nil, ErrInvalid
	}

	return s.repo.ReleaseOverdueRental(ctx, rentalID, jkID)
}

func (s *Service) ExpireRentals(ctx context.Context) error {
	return s.repo.ExpireRentals(ctx)
}

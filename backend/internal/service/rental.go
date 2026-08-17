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

func (s *Service) LockStorage(ctx context.Context, in LockStorageInput) (*models.Rental, error) {
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
	if err != nil || tariffAmount == 0 {
		return nil, fmt.Errorf("service.LockStorage: tariff not set for jk_id %s", st.JKID)
	}

	// Dynamic monthly price calculation
	pricePerMonth := int64(math.Round(st.Area * float64(tariffAmount)))

	// Delegate transactional locking and row creation to the repository
	rt, err := s.repo.LockStorage(ctx, storageID, userID, st.JKID, LockDuration)
	if err != nil {
		return nil, err
	}

	rt.Months = months
	rt.PricePerMonth = pricePerMonth

	if err := s.repo.UpdateRental(ctx, rt); err != nil {
		return nil, fmt.Errorf("service.LockStorage update price info: %w", err)
	}

	return rt, nil
}

func (s *Service) ConfirmPayment(ctx context.Context, in ConfirmPaymentInput) (*models.Rental, error) {
	rentalID := strings.TrimSpace(in.RentalID)
	userID := strings.TrimSpace(in.UserID)

	if rentalID == "" || userID == "" {
		return nil, ErrInvalid
	}

	rt, err := s.repo.GetRentalByID(ctx, rentalID)
	if err != nil {
		return nil, err
	}
	if rt.UserID != userID {
		return nil, ErrForbidden
	}

	now := time.Now().UTC()

	// Check if the lock period has expired
	if rt.Status == "locked" && now.After(rt.EndsAt) {
		return nil, ErrLocked
	}

	ends := now.AddDate(0, rt.Months, 0)

	rt.StartsAt = now
	rt.EndsAt = ends
	rt.TotalPaid = rt.PricePerMonth * int64(rt.Months)
	rt.Status = models.RentalStatusActive

	if err := s.repo.UpdateRental(ctx, rt); err != nil {
		return nil, fmt.Errorf("service.ConfirmPayment update rental: %w", err)
	}

	if err := s.repo.UpdateStorageStatus(ctx, rt.StorageID, models.StatusOccupied); err != nil {
		return nil, fmt.Errorf("service.ConfirmPayment update storage status: %w", err)
	}

	p := &models.Payment{
		RentalID: rentalID,
		UserID:   userID,
		Amount:   rt.TotalPaid,
		Provider: "kaspi",
		Status:   models.PaymentStatusSuccess,
	}
	if err := s.repo.CreatePayment(ctx, p); err != nil {
		return nil, fmt.Errorf("service.ConfirmPayment create payment: %w", err)
	}

	return rt, nil
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

func (s *Service) ListActiveRentalsByJK(ctx context.Context, jkID string) ([]*models.Rental, error) {
	jkID = strings.TrimSpace(jkID)
	if jkID == "" {
		return nil, ErrInvalid
	}

	return s.repo.ListActiveRentalsByJK(ctx, jkID)
}

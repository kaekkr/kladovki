package service

import (
	"context"
	"fmt"
	"time"

	"github.com/kaekkr/kladovki/internal/models"
)

const LockDuration = 2 * time.Minute

func (s *Service) LockStorage(ctx context.Context, storageID, userID string, months int) (*models.Rental, models.PriceQuote, error) {
	_ = s.repo.ExpireLocks(ctx)

	st, err := s.repo.GetStorage(ctx, storageID)
	if err != nil {
		return nil, models.PriceQuote{}, ErrNotFound
	}
	if st.Status != models.StatusFree {
		return nil, models.PriceQuote{}, ErrLocked
	}

	tariff, err := s.repo.GetTariff(ctx, st.JKID)
	if err != nil || tariff == 0 {
		return nil, models.PriceQuote{}, fmt.Errorf("tariff not set")
	}

	if months < 1 {
		months = 1
	}

	quote := CalcPrice(st.Area, tariff, months)
	until := time.Now().Add(LockDuration)

	rt := &models.Rental{
		StorageID:     storageID,
		UserID:        userID,
		JKID:          st.JKID,
		Months:        months,
		PricePerMonth: quote.PricePerMonth,
		TotalPaid:     0,
		LockedUntil:   &until,
		Status:        models.RentalStatusLocked,
	}

	if err := s.repo.CreateRental(ctx, rt); err != nil {
		return nil, models.PriceQuote{}, err
	}

	if err := s.repo.UpdateStorageStatus(ctx, storageID, models.StatusLocked); err != nil {
		return nil, models.PriceQuote{}, err
	}

	return rt, quote, nil
}

func (s *Service) ConfirmPayment(ctx context.Context, rentalID, userID string) (*models.Rental, error) {
	rt, err := s.repo.GetRental(ctx, rentalID)
	if err != nil {
		return nil, ErrNotFound
	}
	if rt.UserID != userID {
		return nil, ErrForbidden
	}
	if rt.Status != models.RentalStatusLocked {
		return nil, ErrInvalid
	}
	if rt.LockedUntil != nil && time.Now().After(*rt.LockedUntil) {
		_ = s.repo.ExpireLocks(ctx)
		return nil, ErrLocked
	}

	now := time.Now()
	ends := now.AddDate(0, rt.Months, 0)
	rt.StartsAt = now
	rt.EndsAt = ends
	rt.TotalPaid = rt.PricePerMonth * int64(rt.Months)
	rt.Status = models.RentalStatusActive
	rt.LockedUntil = nil

	if err := s.repo.UpdateRental(ctx, rt); err != nil {
		return nil, err
	}
	_ = s.repo.UpdateStorageStatus(ctx, rt.StorageID, models.StatusOccupied)

	p := &models.Payment{
		RentalID: rentalID,
		UserID:   userID,
		Amount:   rt.TotalPaid,
		Provider: "kaspi",
		Status:   models.PaymentStatusSuccess,
	}
	_ = s.repo.CreatePayment(ctx, p)

	return rt, nil
}

func (s *Service) ChangeTariff(ctx context.Context, jkID string, newAmount int64) (int64, []models.Rental, error) {
	old, _ := s.repo.GetTariff(ctx, jkID)
	if err := s.repo.SetTariff(ctx, jkID, newAmount); err != nil {
		return 0, nil, err
	}
	rentals, err := s.repo.ListActiveRentalsByJK(ctx, jkID)
	return old, rentals, err
}

func (s *Service) ApplyTariffAction(ctx context.Context, rentalID string, action models.TariffAction, newTariffPerM2 int64) error {
	rt, err := s.repo.GetRental(ctx, rentalID)
	if err != nil {
		return ErrNotFound
	}
	st, err := s.repo.GetStorage(ctx, rt.StorageID)
	if err != nil {
		return err
	}

	oldPerMonth := rt.PricePerMonth
	newPerMonth := CalcPrice(st.Area, newTariffPerM2, 1).PricePerMonth

	remaining := 0.0
	if !rt.EndsAt.IsZero() {
		remaining = rt.EndsAt.Sub(time.Now()).Hours() / (24 * 30)
	}
	if remaining < 0 {
		remaining = 0
	}

	opts := CalcTariffChangeOptions(rt.TotalPaid, oldPerMonth, newPerMonth, remaining)
	if len(opts) == 0 {
		return nil
	}

	switch action {
	case models.TariffActionRefund:
		for _, o := range opts {
			if o.Action == models.TariffActionRefund {
				rt.TotalPaid -= o.RefundAmount
				if rt.TotalPaid < 0 {
					rt.TotalPaid = 0
				}
			}
		}
		rt.PricePerMonth = newPerMonth
	case models.TariffActionKeep:
		for _, o := range opts {
			if o.Action == models.TariffActionKeep && !rt.EndsAt.IsZero() {
				extraDays := int(o.ExtraMonths * 30)
				rt.EndsAt = rt.EndsAt.AddDate(0, 0, extraDays)
				rt.PricePerMonth = newPerMonth
			}
		}
	default:
		return ErrInvalid
	}

	return s.repo.UpdateRental(ctx, rt)
}

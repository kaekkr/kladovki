package service

import (
	"context"
	"fmt"
	"time"

	"github.com/kaekkr/kladovki/internal/models"
)

func (s *Service) GetDashboardStats(
	ctx context.Context,
	jkID string,
	period models.DashboardPeriod,
) (*models.DashboardStats, error) {
	if jkID == "" {
		return nil, ErrInvalid
	}

	now := time.Now().UTC()

	var from time.Time

	switch period {
	case models.DashboardPeriod7Days:
		from = now.AddDate(0, 0, -7)

	case models.DashboardPeriod30Days:
		from = now.AddDate(0, 0, -30)

	case models.DashboardPeriodQuarter:
		from = now.AddDate(0, -3, 0)

	default:
		return nil, ErrInvalid
	}

	duration := now.Sub(from)

	previousFrom := from.Add(-duration)
	previousTo := from

	row, err := s.repo.GetDashboardStats(
		ctx,
		jkID,
		from,
		now,
		previousFrom,
		previousTo,
	)
	if err != nil {
		return nil, fmt.Errorf("service.GetDashboardStats: %w", err)
	}

	var receivedChange float64

	if row.PreviousReceived > 0 {
		receivedChange =
			float64(row.Received-row.PreviousReceived) /
				float64(row.PreviousReceived) *
				100
	} else if row.Received > 0 {
		receivedChange = 100
	}

	return &models.DashboardStats{
		Received:       row.Received,
		ReceivedChange: receivedChange,

		OccupiedStorages: row.OccupiedStorages,
		TotalStorages:    row.TotalStorages,

		ActiveRentals: row.ActiveRentals,
		NewRentals:    row.NewRentals,
	}, nil
}

func (s *Service) GetDashboardRevenue(
	ctx context.Context,
	jkID string,
	period models.DashboardPeriod,
) (*models.DashboardRevenue, error) {
	if jkID == "" {
		return nil, ErrInvalid
	}

	now := time.Now().UTC()

	var from time.Time
	var groupBy string

	switch period {
	case models.DashboardPeriod7Days:
		from = now.AddDate(0, 0, -7)
		groupBy = "day"

	case models.DashboardPeriod30Days:
		from = now.AddDate(0, 0, -30)
		groupBy = "day"

	case models.DashboardPeriodQuarter:
		from = now.AddDate(0, -3, 0)
		groupBy = "week"

	default:
		return nil, ErrInvalid
	}

	rows, err := s.repo.GetDashboardRevenue(
		ctx,
		jkID,
		from,
		now,
		groupBy,
	)
	if err != nil {
		return nil, fmt.Errorf("service.GetDashboardRevenue: %w", err)
	}

	points := make([]models.RevenuePoint, 0, len(rows))

	for _, row := range rows {
		points = append(points, models.RevenuePoint{
			Date:   row.Date.Format("2006-01-02"),
			Amount: row.Amount,
		})
	}

	return &models.DashboardRevenue{
		Points: points,
	}, nil
}

func (s *Service) GetDashboardAttention(
	ctx context.Context,
	jkID string,
) ([]*models.DashboardAttention, error) {
	if jkID == "" {
		return nil, ErrInvalid
	}

	items, err := s.repo.ListDashboardAttention(ctx, jkID)
	if err != nil {
		return nil, fmt.Errorf("service.GetDashboardAttention: %w", err)
	}

	return items, nil
}

func (s *Service) GetDashboardActivity(
	ctx context.Context,
	jkID string,
) ([]*models.DashboardActivity, error) {
	if jkID == "" {
		return nil, ErrInvalid
	}

	activity, err := s.repo.ListDashboardActivity(ctx, jkID, 10)
	if err != nil {
		return nil, fmt.Errorf(
			"service.GetDashboardActivity: %w",
			err,
		)
	}

	return activity, nil
}

func (s *Service) GetDashboardOccupancy(
	ctx context.Context,
	jkID string,
) (*models.DashboardOccupancy, error) {
	if jkID == "" {
		return nil, ErrInvalid
	}

	occupancy, err := s.repo.GetDashboardOccupancy(ctx, jkID)
	if err != nil {
		return nil, fmt.Errorf(
			"service.GetDashboardOccupancy: %w",
			err,
		)
	}

	return occupancy, nil
}

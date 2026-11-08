package service

import (
	"context"
	"errors"
	"strings"

	"github.com/kaekkr/kladovki/internal/models"
	"github.com/kaekkr/kladovki/internal/repository"
)

func mapRepoErr(err error) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return ErrNotFound
	case errors.Is(err, repository.ErrConflict):
		return ErrAlreadyExists
	case errors.Is(err, repository.ErrInvalid):
		return ErrInvalid
	case errors.Is(err, repository.ErrLocked):
		return ErrLocked
	}
	return err
}

func (s *Service) ListRequestsByUser(ctx context.Context, userID string) ([]*models.Request, error) {
	return s.repo.ListRequestsByUser(ctx, userID)
}

func (s *Service) ListRequestsByJK(ctx context.Context, jkID string) ([]*models.Request, error) {
	return s.repo.ListRequestsByJK(ctx, jkID)
}

func (s *Service) CountOpenRequests(ctx context.Context, jkID string) (int64, error) {
	return s.repo.CountOpenRequests(ctx, jkID)
}

func (s *Service) ChooseTariffOption(ctx context.Context, requestID, userID, choice string) error {
	if choice != "extend" && choice != "refund" {
		return ErrInvalid
	}
	return mapRepoErr(s.repo.ApplyTariffChoice(ctx, requestID, userID, choice))
}

func (s *Service) RequestEarlyTermination(ctx context.Context, rentalID, userID, comment string) (*models.Request, error) {
	comment = strings.TrimSpace(comment)
	if len(comment) > 1000 {
		comment = comment[:1000]
	}
	q, err := s.repo.CreateEarlyTermination(ctx, rentalID, userID, comment)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	s.notifyNewRequest(ctx, q)
	return q, nil
}

func (s *Service) ResolveRequest(ctx context.Context, requestID, jkID, adminID, action, resolution string) error {
	resolution = strings.TrimSpace(resolution)
	if err := s.repo.ResolveRequest(ctx, requestID, jkID, adminID, action, resolution); err != nil {
		return mapRepoErr(err)
	}
	s.notifyRequestResolved(ctx, requestID)
	return nil
}

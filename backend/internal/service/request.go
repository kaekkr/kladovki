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
	case errors.Is(err, repository.ErrRefundNotRejectable):
		return ErrRefundNotRejectable
	case errors.Is(err, repository.ErrRenewalPending):
		return ErrRenewalPending
	case errors.Is(err, repository.ErrTerminationPending):
		return ErrTerminationPending
	case errors.Is(err, repository.ErrTariffChoicePending):
		return ErrTariffChoicePending
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
	if err := s.repo.ApplyTariffChoice(ctx, requestID, userID, choice); err != nil {
		if errors.Is(err, repository.ErrTerminationPending) {
			return ErrTariffChoiceTerminating
		}
		return mapRepoErr(err)
	}
	if choice == "refund" {
		if q, err := s.repo.GetRequestByID(ctx, requestID); err == nil {
			s.notifyTariffRefundRequest(ctx, q)
		}
	}
	return nil
}

func (s *Service) RequestEarlyTermination(ctx context.Context, rentalID, userID, comment string) (*models.Request, error) {
	comment = strings.TrimSpace(comment)
	if r := []rune(comment); len(r) > 1000 { // by characters: bytes would split a letter
		comment = string(r[:1000])
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

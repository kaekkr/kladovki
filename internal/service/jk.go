package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kaekkr/kladovki/internal/models"
)

func (s *Service) RegisterJK(ctx context.Context, name, bin, contact, phone, email, password string) (*models.JK, *models.User, error) {
	if _, err := s.repo.GetUserByEmail(ctx, email); err == nil {
		return nil, nil, fmt.Errorf("%w: email already in use", ErrAlreadyExists)
	}
	if _, err := s.repo.GetUserByPhone(ctx, phone); err == nil {
		return nil, nil, fmt.Errorf("%w: phone number already in use", ErrAlreadyExists)
	}
	if _, err := s.repo.GetUserByBIN(ctx, bin); err == nil {
		return nil, nil, fmt.Errorf("%w: BIN already registered", ErrAlreadyExists)
	}

	hash, err := HashPassword(password)
	if err != nil {
		return nil, nil, err
	}

	u := &models.User{
		ID:           uuid.New().String(),
		Roles:        []models.Role{models.RoleAdmin},
		FullName:     contact,
		Phone:        phone,
		Email:        email,
		BIN:          &bin,
		PasswordHash: hash,
		CreatedAt:    time.Now(),
	}

	if err := s.repo.CreateUser(ctx, u); err != nil {
		return nil, nil, err
	}

	jk := &models.JK{
		ID:        uuid.New().String(),
		Name:      name,
		BIN:       bin,
		Contact:   contact,
		Phone:     phone,
		Email:     email,
		OwnerID:   u.ID,
		CreatedAt: time.Now(),
	}

	if err := s.repo.CreateJK(ctx, jk); err != nil {
		return nil, nil, err
	}

	u.JKID = &jk.ID
	_ = s.repo.UpdateUserJKID(ctx, u.ID, jk.ID)

	return jk, u, nil
}

func (s *Service) AddStorages(ctx context.Context, jkID string, items []models.Storage) error {
	for i := range items {
		items[i].JKID = jkID
		items[i].Status = models.StatusFree
		if err := s.repo.CreateStorage(ctx, &items[i]); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) StorageViews(ctx context.Context, jkID string) ([]models.StorageView, error) {
	_ = s.repo.ExpireLocks(ctx)
	storages, err := s.repo.ListStoragesByJK(ctx, jkID)
	if err != nil {
		return nil, err
	}

	views := make([]models.StorageView, 0, len(storages))
	for _, st := range storages {
		v := models.StorageView{
			ID:       st.ID,
			Number:   st.Number,
			Area:     st.Area,
			Floor:    st.Floor,
			Entrance: st.Entrance,
			Status:   st.Status,
			Period:   "-",
			Paid:     0,
			Debt:     0,
		}

		rt, err := s.repo.GetActiveRentalByStorage(ctx, st.ID)
		if err == nil && rt != nil {
			if rt.Status == "active" {
				if !rt.StartsAt.IsZero() && !rt.EndsAt.IsZero() {
					v.Period = fmt.Sprintf("%s — %s",
						rt.StartsAt.Format("02.01.2006"),
						rt.EndsAt.Format("02.01.2006"))
				}
				v.Paid = rt.TotalPaid

				if !rt.EndsAt.IsZero() && time.Now().After(rt.EndsAt) {
					days := int(time.Since(rt.EndsAt).Hours() / 24)
					v.Debt = rt.PricePerMonth
					v.DaysOver = days
				}
			} else if rt.Status == "locked" {
				v.Status = models.StatusLocked
			}
		}
		views = append(views, v)
	}
	return views, nil
}

package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kaekkr/kladovki/internal/models"
)

func (s *Service) EnsureSeed(ctx context.Context) error {
	_, err := s.repo.GetJKByName(ctx, "ЖК Сыганак")
	if err == nil {
		return nil // Already seeded
	}

	hash, err := HashPassword("admin123")
	if err != nil {
		return fmt.Errorf("hash password error: %w", err)
	}

	bin := "123456789012"
	adminID := uuid.New().String()
	jkID := uuid.New().String()
	now := time.Now()

	// 1. Create Admin User (FK to JKID will be populated right after)
	u := &models.User{
		ID:           adminID,
		Roles:        []models.Role{models.RoleAdmin}, // Updated to slice
		FullName:     "Админ Сыганак",
		Phone:        "+77001112233",
		Email:        "admin@syganak.kz",
		BIN:          &bin,
		PasswordHash: hash,
		CreatedAt:    now,
	}
	if err := s.repo.CreateUser(ctx, u); err != nil {
		return fmt.Errorf("seed admin user: %w", err)
	}

	// 2. Create JK
	jk := &models.JK{
		ID:        jkID,
		Name:      "ЖК Сыганак",
		BIN:       bin,
		Contact:   "Админ",
		Phone:     u.Phone,
		Email:     u.Email,
		OwnerID:   adminID,
		CreatedAt: now,
	}
	if err := s.repo.CreateJK(ctx, jk); err != nil {
		return fmt.Errorf("seed jk: %w", err)
	}

	// 3. Link Admin to JK
	u.JKID = &jkID
	if err := s.repo.UpdateUserJKID(ctx, adminID, jkID); err != nil {
		return fmt.Errorf("update admin jkid: %w", err)
	}

	// 4. Set Initial Tariff
	if err := s.repo.SetTariff(ctx, jkID, 15000); err != nil {
		return fmt.Errorf("seed tariff: %w", err)
	}

	// 5. Seed Demo Storages
	demo := []models.Storage{
		{Number: "A-01", Area: 4.5, Floor: -1, Entrance: 1},
		{Number: "A-02", Area: 5.0, Floor: -1, Entrance: 1},
		{Number: "A-03", Area: 3.8, Floor: -1, Entrance: 1},
		{Number: "A-04", Area: 6.2, Floor: -1, Entrance: 1},
		{Number: "B-01", Area: 4.0, Floor: -1, Entrance: 2},
		{Number: "B-02", Area: 5.5, Floor: -1, Entrance: 2},
		{Number: "B-03", Area: 4.8, Floor: -1, Entrance: 2},
		{Number: "C-01", Area: 7.0, Floor: -1, Entrance: 3},
		{Number: "C-02", Area: 3.5, Floor: -1, Entrance: 3},
		{Number: "C-03", Area: 5.2, Floor: -1, Entrance: 3},
		{Number: "D-01", Area: 4.2, Floor: -1, Entrance: 4},
		{Number: "D-02", Area: 6.0, Floor: -1, Entrance: 4},
	}

	for i := range demo {
		demo[i].ID = uuid.New().String()
		demo[i].JKID = jkID
		demo[i].Status = models.StatusFree
		demo[i].CreatedAt = now
		if err := s.repo.CreateStorage(ctx, &demo[i]); err != nil {
			return fmt.Errorf("seed storage %s: %w", demo[i].Number, err)
		}
	}

	return nil
}

// internal/repository/contract.go
package repository

import (
	"context"

	"github.com/kaekkr/kladovki/internal/models"
)

type UserRepository interface {
	CreateUser(ctx context.Context, u *models.User) error
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	GetUserByPhone(ctx context.Context, phone string) (*models.User, error)
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	UpdateUserJKID(ctx context.Context, userID, jkID string) error
}

package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/toanehihi/object-storage/internal/model"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository interface {
	CreateUser(ctx context.Context, user *model.User) error
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.User, error)
}

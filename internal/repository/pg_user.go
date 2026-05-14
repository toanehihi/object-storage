package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/toanehihi/object-storage/internal/model"
	"github.com/toanehihi/object-storage/sqlc"
)

type PgUserRepository struct {
	q *sqlc.Queries
}

func NewPgUserRepository(pool *pgxpool.Pool) *PgUserRepository {
	return &PgUserRepository{q: sqlc.New(pool)}
}

func (r *PgUserRepository) CreateUser(ctx context.Context, user *model.User) error {
	return r.q.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           user.ID,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		FullName:     pgtype.Text{String: user.FullName, Valid: user.FullName != ""},
		AvatarUrl:    pgtype.Text{String: user.AvatarURL, Valid: user.AvatarURL != ""},
		CreatedAt:    pgtype.Timestamp{Time: user.CreatedAt, Valid: true},
		UpdatedAt:    pgtype.Timestamp{Time: user.UpdatedAt, Valid: true},
	})
}

func (r *PgUserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return toUserModel(row), nil
}

func (r *PgUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return toUserModel(row), nil
}

func toUserModel(u sqlc.User) *model.User {
	return &model.User{
		ID:           u.ID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		FullName:     u.FullName.String,
		AvatarURL:    u.AvatarUrl.String,
		CreatedAt:    u.CreatedAt.Time,
		UpdatedAt:    u.UpdatedAt.Time,
	}
}

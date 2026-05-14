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

type PgFileRepository struct {
	q *sqlc.Queries
}

func NewPgFileRepository(pool *pgxpool.Pool) *PgFileRepository {
	return &PgFileRepository{q: sqlc.New(pool)}
}

func (r *PgFileRepository) CreateFile(ctx context.Context, file *model.File) error {
	return r.q.CreateFile(ctx, sqlc.CreateFileParams{
		ID:          file.ID,
		OwnerID:     file.OwnerID,
		Filename:    file.Filename,
		ObjectKey:   file.ObjectKey,
		Size:        file.Size,
		ContentType: pgtype.Text{String: file.ContentType, Valid: file.ContentType != ""},
		Status:      string(file.Status),
		Checksum:    pgtype.Text{String: file.Checksum, Valid: file.Checksum != ""},
		CreatedAt:   pgtype.Timestamp{Time: file.CreatedAt, Valid: true},
		UpdatedAt:   pgtype.Timestamp{Time: file.CreatedAt, Valid: true},
	})
}

func (r *PgFileRepository) GetByID(ctx context.Context, fileID uuid.UUID) (*model.File, error) {
	row, err := r.q.GetFileByID(ctx, fileID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrFileNotFound
	}
	if err != nil {
		return nil, err
	}
	return toFileModel(row), nil
}

func (r *PgFileRepository) GetByIDAndOwner(ctx context.Context, fileID, ownerID uuid.UUID) (*model.File, error) {
	row, err := r.q.GetFileByIDAndOwner(ctx, sqlc.GetFileByIDAndOwnerParams{
		ID:      fileID,
		OwnerID: ownerID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrFileNotFound
	}
	if err != nil {
		return nil, err
	}
	return toFileModel(row), nil
}

func (r *PgFileRepository) UpdateStatus(ctx context.Context, fileID uuid.UUID, status model.FileStatus) error {
	return r.q.UpdateFileStatus(ctx, sqlc.UpdateFileStatusParams{
		ID:     fileID,
		Status: string(status),
	})
}

func (r *PgFileRepository) SoftDelete(ctx context.Context, fileID, ownerID uuid.UUID) error {
	return r.q.SoftDeleteFile(ctx, sqlc.SoftDeleteFileParams{
		ID:      fileID,
		OwnerID: ownerID,
	})
}

func (r *PgFileRepository) ListByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int32) ([]*model.File, error) {
	rows, err := r.q.ListFilesByOwner(ctx, sqlc.ListFilesByOwnerParams{
		OwnerID: ownerID,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, err
	}

	files := make([]*model.File, 0, len(rows))
	for _, row := range rows {
		files = append(files, toFileModel(row))
	}
	return files, nil
}

func toFileModel(f sqlc.File) *model.File {
	return &model.File{
		ID:          f.ID,
		OwnerID:     f.OwnerID,
		Filename:    f.Filename,
		ObjectKey:   f.ObjectKey,
		Size:        f.Size,
		ContentType: f.ContentType.String,
		Status:      model.FileStatus(f.Status),
		Checksum:    f.Checksum.String,
		CreatedAt:   f.CreatedAt.Time,
		UpdatedAt:   f.UpdatedAt.Time,
	}
}

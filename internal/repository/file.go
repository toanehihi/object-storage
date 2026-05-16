package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/toanehihi/object-storage/internal/model"
)

var ErrFileNotFound = errors.New("file not found")

type FileRepository interface {
	CreateFile(ctx context.Context, file *model.File) error
	GetByID(ctx context.Context, fileID uuid.UUID) (*model.File, error)
	GetByIDAndOwner(ctx context.Context, fileID, ownerID uuid.UUID) (*model.File, error)
	UpdateStatus(ctx context.Context, fileID uuid.UUID, status model.FileStatus) error
	UpdateScanResult(ctx context.Context, fileID uuid.UUID, status model.FileStatus, scanResult string, scannedAt time.Time) error
	SoftDelete(ctx context.Context, fileID, ownerID uuid.UUID) error
	ListByOwner(ctx context.Context, ownerID uuid.UUID, limit, offset int32) ([]*model.File, error)
	CountByOwner(ctx context.Context, ownerID uuid.UUID) (int64, error)
}

type ChunkRepository interface {
	CreateChunk(ctx context.Context, chunk *model.FileChunk) error
	MarkUploaded(ctx context.Context, fileID uuid.UUID, chunkIndex int32, etag string) error
	GetUploadedIndices(ctx context.Context, fileID uuid.UUID) ([]int32, error)
	CountTotal(ctx context.Context, fileID uuid.UUID) (int32, error)
	CountUploaded(ctx context.Context, fileID uuid.UUID) (int32, error)
}

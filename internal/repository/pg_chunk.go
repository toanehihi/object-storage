package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/toanehihi/object-storage/internal/model"
	"github.com/toanehihi/object-storage/sqlc"
)

type PgChunkRepository struct {
	q *sqlc.Queries
}

func NewPgChunkRepository(pool *pgxpool.Pool) *PgChunkRepository {
	return &PgChunkRepository{q: sqlc.New(pool)}
}

func (r *PgChunkRepository) CreateChunk(ctx context.Context, chunk *model.FileChunk) error {
	return r.q.CreateChunk(ctx, sqlc.CreateChunkParams{
		ID:         chunk.ID,
		FileID:     chunk.FileID,
		ChunkIndex: int32(chunk.ChunkIndex),
		Uploaded:   pgtype.Bool{Bool: chunk.Uploaded, Valid: true},
		CreatedAt:  pgtype.Timestamp{Time: chunk.CreatedAt, Valid: true},
	})
}

func (r *PgChunkRepository) MarkUploaded(ctx context.Context, fileID uuid.UUID, chunkIndex int32, etag string) error {
	return r.q.MarkChunkUploaded(ctx, sqlc.MarkChunkUploadedParams{
		FileID:     fileID,
		ChunkIndex: chunkIndex,
		Etag:       pgtype.Text{String: etag, Valid: etag != ""},
	})
}

func (r *PgChunkRepository) GetUploadedIndices(ctx context.Context, fileID uuid.UUID) ([]int32, error) {
	return r.q.GetUploadedChunkIndices(ctx, fileID)
}

func (r *PgChunkRepository) CountTotal(ctx context.Context, fileID uuid.UUID) (int32, error) {
	return r.q.CountChunksByFileID(ctx, fileID)
}

func (r *PgChunkRepository) CountUploaded(ctx context.Context, fileID uuid.UUID) (int32, error) {
	return r.q.CountUploadedChunksByFileID(ctx, fileID)
}

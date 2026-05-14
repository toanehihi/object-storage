package service

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
	miniogo "github.com/minio/minio-go/v7"

	"github.com/toanehihi/object-storage/internal/model"
	"github.com/toanehihi/object-storage/internal/repository"
	"github.com/toanehihi/object-storage/pkg/minio"
)

var (
	ErrUploadIncomplete = errors.New("not all chunks have been uploaded")
	ErrInvalidChunkSize = errors.New("chunk size must be greater than zero")
)

type InitUploadRequest struct {
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	ContentType string `json:"contentType"`
	ChunkSize   int64  `json:"chunkSize"`
}

type InitUploadResponse struct {
	FileID         uuid.UUID `json:"fileId"`
	ObjectKey      string    `json:"objectKey"`
	ChunkSize      int64     `json:"chunkSize"`
	TotalChunks    int       `json:"totalChunks"`
	PresignedChunk []string  `json:"presignedChunk"`
}

type ChunkCompleteRequest struct {
	FileID     uuid.UUID `json:"fileId"`
	ChunkIndex int32     `json:"chunkIndex"`
	ETag       string    `json:"etag"`
}

type UploadStatusResponse struct {
	FileID         uuid.UUID `json:"fileId"`
	Status         string    `json:"status"`
	UploadedChunks []int32   `json:"uploadedChunks"`
	TotalChunks    int32     `json:"totalChunks"`
}

type UploadService struct {
	fileRepo  repository.FileRepository
	chunkRepo repository.ChunkRepository
	client    *miniogo.Client
	bucket    string
}

func NewUploadService(fileRepo repository.FileRepository, chunkRepo repository.ChunkRepository, client *miniogo.Client, bucket string) *UploadService {
	return &UploadService{
		fileRepo:  fileRepo,
		chunkRepo: chunkRepo,
		client:    client,
		bucket:    bucket,
	}
}

func (s *UploadService) InitUpload(ctx context.Context, ownerID uuid.UUID, req InitUploadRequest) (*InitUploadResponse, error) {
	if req.ChunkSize <= 0 {
		return nil, ErrInvalidChunkSize
	}

	fileID := uuid.New()
	objectKey := "uploads/" + fileID.String()
	totalChunks := int((req.Size + req.ChunkSize - 1) / req.ChunkSize)
	now := time.Now().UTC()

	file := &model.File{
		ID:          fileID,
		OwnerID:     ownerID,
		Filename:    req.Filename,
		ObjectKey:   objectKey,
		Size:        req.Size,
		ContentType: req.ContentType,
		Status:      model.FileStatusUploading,
		CreatedAt:   now,
	}

	if err := s.fileRepo.CreateFile(ctx, file); err != nil {
		return nil, err
	}

	presignedChunks := make([]string, totalChunks)

	for i := range totalChunks {
		chunk := &model.FileChunk{
			ID:         uuid.New(),
			FileID:     fileID,
			ChunkIndex: i,
			Uploaded:   false,
			CreatedAt:  now,
		}
		if err := s.chunkRepo.CreateChunk(ctx, chunk); err != nil {
			return nil, err
		}

		chunkObjectKey := objectKey + "/" + strconv.Itoa(i)
		url, err := minio.GenerateUploadURL(ctx, s.client, s.bucket, chunkObjectKey, 10*time.Minute)
		if err != nil {
			return nil, err
		}
		presignedChunks[i] = url
	}


	return &InitUploadResponse{
		FileID:      fileID,
		ObjectKey:   objectKey,
		ChunkSize:   req.ChunkSize,
		TotalChunks: totalChunks,
		PresignedChunk: presignedChunks,
	}, nil
}

func (s *UploadService) MarkChunkComplete(ctx context.Context, req ChunkCompleteRequest) error {
	return s.chunkRepo.MarkUploaded(ctx, req.FileID, req.ChunkIndex, req.ETag)
}

func (s *UploadService) GetStatus(ctx context.Context, fileID uuid.UUID) (*UploadStatusResponse, error) {
	file, err := s.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return nil, err
	}

	uploaded, err := s.chunkRepo.GetUploadedIndices(ctx, fileID)
	if err != nil {
		return nil, err
	}

	total, err := s.chunkRepo.CountTotal(ctx, fileID)
	if err != nil {
		return nil, err
	}

	return &UploadStatusResponse{
		FileID:         fileID,
		Status:         string(file.Status),
		UploadedChunks: uploaded,
		TotalChunks:    total,
	}, nil
}

func (s *UploadService) CompleteUpload(ctx context.Context, fileID uuid.UUID) error {
	total, err := s.chunkRepo.CountTotal(ctx, fileID)
	if err != nil {
		return err
	}

	uploaded, err := s.chunkRepo.CountUploaded(ctx, fileID)
	if err != nil {
		return err
	}

	if uploaded != total {
		return ErrUploadIncomplete
	}

	return s.fileRepo.UpdateStatus(ctx, fileID, model.FileStatusProcessing)
}

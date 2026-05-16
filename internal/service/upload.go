package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	miniogo "github.com/minio/minio-go/v7"
	"github.com/nats-io/nats.go"

	"github.com/toanehihi/object-storage/internal/model"
	"github.com/toanehihi/object-storage/internal/repository"
	"github.com/toanehihi/object-storage/pkg/minio"
)

var (
	ErrUploadIncomplete     = errors.New("not all chunks have been uploaded")
	ErrInvalidChunkSize     = errors.New("chunk size must be greater than zero")
	ErrChunkIndexOutOfRange = errors.New("chunk index out of range")
	ErrFileNotOwned         = errors.New("file not found or not owned by user")
)

type InitUploadRequest struct {
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	ContentType string `json:"contentType"`
	ChunkSize   int64  `json:"chunkSize"`
}

type InitUploadResponse struct {
	FileID      uuid.UUID `json:"fileId"`
	ObjectKey   string    `json:"objectKey"`
	ChunkSize   int64     `json:"chunkSize"`
	TotalChunks int       `json:"totalChunks"`
}

type ChunkUploadURLResponse struct {
	ChunkIndex int    `json:"chunkIndex"`
	URL        string `json:"url"`
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
	fileRepo    repository.FileRepository
	chunkRepo   repository.ChunkRepository
	client      *miniogo.Client
	bucket      string
	js          nats.JetStreamContext
	maxScanSize int64
}

func NewUploadService(
	fileRepo repository.FileRepository,
	chunkRepo repository.ChunkRepository,
	client *miniogo.Client,
	bucket string,
	js nats.JetStreamContext,
	maxScanSize int64,
) *UploadService {
	return &UploadService{
		fileRepo:    fileRepo,
		chunkRepo:   chunkRepo,
		client:      client,
		bucket:      bucket,
		js:          js,
		maxScanSize: maxScanSize,
	}
}

type FileUploadedEvent struct {
	FileID      string `json:"file_id"`
	ObjectKey   string `json:"object_key"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
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
	}

	return &InitUploadResponse{
		FileID:      fileID,
		ObjectKey:   objectKey,
		ChunkSize:   req.ChunkSize,
		TotalChunks: totalChunks,
	}, nil
}

func (s *UploadService) GetChunkUploadURL(ctx context.Context, ownerID, fileID uuid.UUID, chunkIndex int) (*ChunkUploadURLResponse, error) {
	file, err := s.fileRepo.GetByIDAndOwner(ctx, fileID, ownerID)
	if err != nil {
		if errors.Is(err, repository.ErrFileNotFound) {
			return nil, ErrFileNotOwned
		}
		return nil, err
	}

	totalChunks, err := s.chunkRepo.CountTotal(ctx, fileID)
	if err != nil {
		return nil, err
	}

	if chunkIndex < 0 || chunkIndex >= int(totalChunks) {
		return nil, fmt.Errorf("%w: got %d, total %d", ErrChunkIndexOutOfRange, chunkIndex, totalChunks)
	}

	chunkObjectKey := file.ObjectKey + "/" + strconv.Itoa(chunkIndex)
	url, err := minio.GenerateUploadURL(ctx, s.client, s.bucket, chunkObjectKey, 10*time.Minute)
	if err != nil {
		return nil, err
	}

	return &ChunkUploadURLResponse{
		ChunkIndex: chunkIndex,
		URL:        url,
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

	// Get the file record to know the object key
	file, err := s.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return err
	}

	// Compose all chunk objects into a single final object
	srcs := make([]miniogo.CopySrcOptions, int(total))
	for i := range int(total) {
		srcs[i] = miniogo.CopySrcOptions{
			Bucket: s.bucket,
			Object: file.ObjectKey + "/" + strconv.Itoa(i),
		}
	}

	dst := miniogo.CopyDestOptions{
		Bucket: s.bucket,
		Object: file.ObjectKey,
	}

	if _, err := s.client.ComposeObject(ctx, dst, srcs...); err != nil {
		return fmt.Errorf("failed to compose chunks: %w", err)
	}

	// Clean up individual chunk objects
	for i := range int(total) {
		chunkKey := file.ObjectKey + "/" + strconv.Itoa(i)
		_ = s.client.RemoveObject(ctx, s.bucket, chunkKey, miniogo.RemoveObjectOptions{})
	}

	// Set status to UPLOADED after successful chunk composition
	if err := s.fileRepo.UpdateStatus(ctx, fileID, model.FileStatusUploaded); err != nil {
		return err
	}

	// Skip scan for files exceeding the size limit
	if file.Size > s.maxScanSize {
		return s.fileRepo.UpdateStatus(ctx, fileID, model.FileStatusUnscanned)
	}

	// Publish scan event
	event := FileUploadedEvent{
		FileID:      fileID.String(),
		ObjectKey:   file.ObjectKey,
		Size:        file.Size,
		ContentType: file.ContentType,
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if _, err := s.js.Publish("file.uploaded", data); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

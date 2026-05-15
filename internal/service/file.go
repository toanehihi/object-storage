package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	miniogo "github.com/minio/minio-go/v7"

	"github.com/toanehihi/object-storage/internal/model"
	"github.com/toanehihi/object-storage/internal/repository"
	"github.com/toanehihi/object-storage/pkg/minio"
)

var ErrFileNotReady = errors.New("file is not ready for download")

type DownloadURLResponse struct {
	FileID      uuid.UUID `json:"fileId"`
	Filename    string    `json:"filename"`
	DownloadURL string    `json:"downloadUrl"`
	ExpiresIn   int       `json:"expiresIn"`
}

type FileService struct {
	fileRepo repository.FileRepository
	client   *miniogo.Client
	bucket   string
}

func NewFileService(fileRepo repository.FileRepository, client *miniogo.Client, bucket string) *FileService {
	return &FileService{
		fileRepo: fileRepo,
		client:   client,
		bucket:   bucket,
	}
}

func (s *FileService) GetMetadata(ctx context.Context, fileID, ownerID uuid.UUID) (*model.File, error) {
	return s.fileRepo.GetByIDAndOwner(ctx, fileID, ownerID)
}

func (s *FileService) Delete(ctx context.Context, fileID, ownerID uuid.UUID) error {
	return s.fileRepo.SoftDelete(ctx, fileID, ownerID)
}

func (s *FileService) ListFiles(ctx context.Context, ownerID uuid.UUID, limit, offset int32) ([]*model.File, error) {
	return s.fileRepo.ListByOwner(ctx, ownerID, limit, offset)
}

func (s *FileService) GetDownloadURL(ctx context.Context, fileID, ownerID uuid.UUID) (*DownloadURLResponse, error) {
	file, err := s.fileRepo.GetByIDAndOwner(ctx, fileID, ownerID)
	if err != nil {
		return nil, err
	}

	if file.Status != model.FileStatusReady {
		return nil, ErrFileNotReady
	}

	expiry := 1 * time.Hour
	url, err := minio.GenerateDownloadURL(ctx, s.client, s.bucket, file.ObjectKey, file.Filename, expiry)
	if err != nil {
		return nil, err
	}

	return &DownloadURLResponse{
		FileID:      file.ID,
		Filename:    file.Filename,
		DownloadURL: url,
		ExpiresIn:   int(expiry.Seconds()),
	}, nil
}

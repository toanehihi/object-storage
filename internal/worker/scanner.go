package worker

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	clamd "github.com/dutchcoders/go-clamd"
	"github.com/google/uuid"
	miniogo "github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/toanehihi/object-storage/internal/model"
	"github.com/toanehihi/object-storage/internal/repository"
)

type ScanResult string

const (
	ScanResultClean    ScanResult = "CLEAN"
	ScanResultInfected ScanResult = "INFECTED"
)

type Scanner struct {
	clam     *clamd.Clamd
	minio    *miniogo.Client
	bucket   string
	fileRepo repository.FileRepository
	logger   *zap.Logger
	timeout  time.Duration
}

func NewScanner(
	clamAddr string,
	minio *miniogo.Client,
	bucket string,
	fileRepo repository.FileRepository,
	logger *zap.Logger,
	timeout time.Duration,
) *Scanner {
	return &Scanner{
		clam:     clamd.NewClamd("tcp://" + clamAddr),
		minio:    minio,
		bucket:   bucket,
		fileRepo: fileRepo,
		logger:   logger,
		timeout:  timeout,
	}
}

func (s *Scanner) Scan(ctx context.Context, fileID uuid.UUID, objectKey string) error {
	if err := s.fileRepo.UpdateStatus(ctx, fileID, model.FileStatusProcessing); err != nil {
		return fmt.Errorf("failed to set PROCESSING: %w", err)
	}

	obj, err := s.minio.GetObject(ctx, s.bucket, objectKey, miniogo.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to get object from MinIO: %w", err)
	}
	defer obj.Close()

	scanCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	abortChan := make(chan bool, 1)
	go func() {
		<-scanCtx.Done()
		abortChan <- true
	}()

	response, err := s.clam.ScanStream(obj, abortChan)
	if err != nil {
		return fmt.Errorf("clamd scan failed: %w", err)
	}

	for result := range response {
		if strings.HasPrefix(result.Status, "FOUND") {
			s.logger.Warn("virus detected",
				zap.String("file_id", fileID.String()),
				zap.String("virus", result.Description),
			)

			removeErr := s.minio.RemoveObject(ctx, s.bucket, objectKey, miniogo.RemoveObjectOptions{})
			if removeErr != nil {
				s.logger.Error("failed to delete infected object",
					zap.String("file_id", fileID.String()),
					zap.Error(removeErr),
				)
			}

			return s.fileRepo.UpdateScanResult(ctx, fileID, model.FileStatusInfected, string(ScanResultInfected), time.Now().UTC())
		}

		if result.Status == "ERROR" {
			return fmt.Errorf("clamd returned error: %s", result.Description)
		}
	}

	s.logger.Info("file clean", zap.String("file_id", fileID.String()))
	return s.fileRepo.UpdateScanResult(ctx, fileID, model.FileStatusReady, string(ScanResultClean), time.Now().UTC())
}

func (s *Scanner) Ping() error {
	return s.clam.Ping()
}

// DrainReader reads and discards remaining data from r to prevent broken pipe in clamd.
func DrainReader(r io.Reader) {
	_, _ = io.Copy(io.Discard, r)
}

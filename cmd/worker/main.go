package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/toanehihi/object-storage/config"
	"github.com/toanehihi/object-storage/internal/repository"
	"github.com/toanehihi/object-storage/internal/service"
	"github.com/toanehihi/object-storage/internal/worker"
	pkgminio "github.com/toanehihi/object-storage/pkg/minio"
	pkgnats "github.com/toanehihi/object-storage/pkg/nats"
	pkgpg "github.com/toanehihi/object-storage/pkg/postgres"
)

func main() {
	logger, _ := zap.NewProduction()
	defer func() { _ = logger.Sync() }()

	if err := run(logger); err != nil {
		log.Fatal(err)
	}
}

func run(logger *zap.Logger) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	pool, err := pkgpg.NewPool(ctx, cfg.Postgres.DSN())
	if err != nil {
		return err
	}
	defer pool.Close()
	logger.Info("connected to PostgreSQL")

	minioClient, err := pkgminio.NewClient(pkgminio.ClientConfig{
		Endpoint:  cfg.MinIO.Endpoint,
		AccessKey: cfg.MinIO.AccessKey,
		SecretKey: cfg.MinIO.SecretKey,
		UseSSL:    cfg.MinIO.UseSSL,
	})
	if err != nil {
		return err
	}
	logger.Info("connected to MinIO")

	natsConn, err := pkgnats.NewConn(cfg.NATS.URL, logger)
	if err != nil {
		return err
	}
	defer natsConn.Close()
	logger.Info("connected to NATS")

	js, err := natsConn.JetStream()
	if err != nil {
		return err
	}

	// Ensure stream exists
	_, err = js.AddStream(&nats.StreamConfig{
		Name:      "FILE_EVENTS",
		Subjects:  []string{"file.uploaded"},
		Retention: nats.WorkQueuePolicy,
		Storage:   nats.FileStorage,
	})
	if err != nil {
		return err
	}
	logger.Info("ensured JetStream stream FILE_EVENTS")

	fileRepo := repository.NewPgFileRepository(pool)

	scanner := worker.NewScanner(
		cfg.ClamAV.Address,
		minioClient,
		cfg.MinIO.Bucket,
		fileRepo,
		logger,
		cfg.ClamAV.ScanTimeout,
	)

	if err := scanner.Ping(); err != nil {
		logger.Warn("clamd not reachable at startup, will retry on messages", zap.Error(err))
	} else {
		logger.Info("clamd is reachable", zap.String("address", cfg.ClamAV.Address))
	}

	// Subscribe with durable consumer
	sub, err := js.Subscribe("file.uploaded", func(msg *nats.Msg) {
		var event service.FileUploadedEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			logger.Error("failed to unmarshal event", zap.Error(err))
			_ = msg.Term()
			return
		}

		fileID, err := uuid.Parse(event.FileID)
		if err != nil {
			logger.Error("invalid file ID in event", zap.String("file_id", event.FileID), zap.Error(err))
			_ = msg.Term()
			return
		}

		logger.Info("scanning file",
			zap.String("file_id", event.FileID),
			zap.String("object_key", event.ObjectKey),
			zap.Int64("size", event.Size),
		)

		if err := scanner.Scan(ctx, fileID, event.ObjectKey); err != nil {
			logger.Error("scan failed, will retry",
				zap.String("file_id", event.FileID),
				zap.Error(err),
			)
			_ = msg.NakWithDelay(30 * time.Second)
			return
		}

		_ = msg.Ack()
	}, nats.Durable("clamav-scanner"), nats.AckWait(5*time.Minute), nats.MaxAckPending(1))
	if err != nil {
		return err
	}

	logger.Info("worker started, waiting for messages")

	// Graceful Shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
	<-shutdown

	logger.Info("shutting down worker...")
	_ = sub.Unsubscribe()

	return nil
}

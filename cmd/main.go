package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/toanehihi/object-storage/config"
	"github.com/toanehihi/object-storage/internal/handler"
	"github.com/toanehihi/object-storage/internal/middleware"
	"github.com/toanehihi/object-storage/internal/repository"
	"github.com/toanehihi/object-storage/internal/service"
	"github.com/toanehihi/object-storage/internal/token"
	"github.com/toanehihi/object-storage/internal/utils/response"
	pkgminio "github.com/toanehihi/object-storage/pkg/minio"
	pkgnats "github.com/toanehihi/object-storage/pkg/nats"
	pkgpg "github.com/toanehihi/object-storage/pkg/postgres"
)

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
	Version string `json:"version"`
}

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

	// --- Infrastructure ---

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

	if err := pkgminio.EnsureBucketExists(ctx, minioClient, cfg.MinIO.Bucket); err != nil {
		return err
	}
	logger.Info("ensured MinIO bucket exists", zap.String("bucket", cfg.MinIO.Bucket))

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

	// Ensure stream exists here as well so the API server can publish even if worker starts later
	_, err = js.AddStream(&nats.StreamConfig{
		Name:      "FILE_EVENTS",
		Subjects:  []string{"file.uploaded"},
		Retention: nats.WorkQueuePolicy,
		Storage:   nats.FileStorage,
	})
	if err != nil {
		return err
	}
	logger.Info("connected to NATS JetStream and ensured FILE_EVENTS stream exists")

	// --- Repositories ---

	userRepo := repository.NewPgUserRepository(pool)
	fileRepo := repository.NewPgFileRepository(pool)
	chunkRepo := repository.NewPgChunkRepository(pool)

	// --- Services ---

	tokenMgr := token.NewManager(cfg.JWT.Secret, cfg.JWT.Expiry)
	authSvc := service.NewAuthService(userRepo, tokenMgr)
	uploadSvc := service.NewUploadService(fileRepo, chunkRepo, minioClient, cfg.MinIO.Bucket, js, cfg.ClamAV.MaxScanSize)
	fileSvc := service.NewFileService(fileRepo, minioClient, cfg.MinIO.Bucket)

	// --- Handlers ---

	authHandler := handler.NewAuthHandler(authSvc)
	uploadHandler := handler.NewUploadHandler(uploadSvc)
	fileHandler := handler.NewFileHandler(fileSvc)
	authMW := middleware.JWTAuth(tokenMgr)

	// --- Echo Server ---

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.Use(echomw.RequestID())
	e.Use(echomw.Recover())
	e.Use(echomw.Logger())
	e.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins: []string{"http://localhost:3000"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowMethods: []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.OPTIONS},
	}))

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, response.OK(HealthResponse{
			Status:  "ok",
			Service: "object-storage",
			Version: "1.0.0",
		}, ""))
	})

	v1 := e.Group("/api/v1")

	authGroup := v1.Group("/auth")
	authHandler.RegisterRoutes(authGroup, authMW)

	uploadGroup := v1.Group("/uploads", authMW)
	uploadHandler.RegisterRoutes(uploadGroup)

	fileGroup := v1.Group("/files", authMW)
	fileHandler.RegisterRoutes(fileGroup)

	// --- Graceful Shutdown ---

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		addr := ":" + strconv.Itoa(cfg.Server.Port)
		logger.Info("server starting", zap.String("addr", addr))
		if err := e.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	<-shutdown
	logger.Info("shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		return err
	}

	logger.Info("server stopped")
	return nil
}

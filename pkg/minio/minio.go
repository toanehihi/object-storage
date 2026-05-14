package minio

import (
	"context"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type ClientConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

func NewClient(cfg ClientConfig) (*minio.Client, error) {
	return minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
}

func GenerateUploadURL(ctx context.Context, client *minio.Client, bucket, object string, expiry time.Duration) (string, error) {
	url, err := client.PresignedPutObject(ctx, bucket, object, expiry)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

func GenerateDownloadURL(ctx context.Context, client *minio.Client, bucket, object string, 	expiry time.Duration) (string, error) {
	url, err := client.PresignedGetObject(ctx, bucket, object, expiry, nil)

	if err != nil {
		return "", err
	}
	return url.String(), nil
}

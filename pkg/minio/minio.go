package minio

import (
	"context"
	"fmt"
	"net/url"
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

func EnsureBucketExists(ctx context.Context, client *minio.Client, bucket string) error {
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return err
	}
	if !exists {
		return client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
	}
	return nil
}

func GenerateUploadURL(ctx context.Context, client *minio.Client, bucket, object string, expiry time.Duration) (string, error) {
	url, err := client.PresignedPutObject(ctx, bucket, object, expiry)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

func GenerateDownloadURL(ctx context.Context, client *minio.Client, bucket, object string, filename string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	reqParams.Set("response-content-disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))

	url, err := client.PresignedGetObject(ctx, bucket, object, expiry, reqParams)
	if err != nil {
		return "", err
	}
	return url.String(), nil
}

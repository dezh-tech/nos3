package minio

import (
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func New(cfg *ClientConfig) (*minio.Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:           credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:          false,
		TrailingHeaders: true,
	})

	return client, err
}

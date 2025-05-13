package minio

import (
	"context"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	grpcRepository "nos3/internal/domain/repository/grpcclient"
	"nos3/pkg/logger"
)

type Client struct {
	MinioClient *minio.Client
	GrpcClient  grpcRepository.IClient
}

func New(cfg *ClientConfig, grpcClient grpcRepository.IClient) (*Client, error) {
	logger.Info("connecting to minio")

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:           credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure:          false,
		TrailingHeaders: true,
	})
	if err != nil {
		if _, logErr := grpcClient.AddLog(context.Background(),
			"failed to initialize MinIO client", err.Error()); logErr != nil {
			logger.Error("can't send log to manager", "err", logErr)
		}

		return nil, err
	}

	return &Client{
		MinioClient: client,
		GrpcClient:  grpcClient,
	}, nil
}

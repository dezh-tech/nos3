package minio

import (
	"context"
	"time"

	"github.com/minio/minio-go/v7"

	grpcRepository "nos3/internal/domain/repository/grpcclient"
	"nos3/pkg/logger"
)

type Remover struct {
	minioClient *minio.Client
	grpcClient  grpcRepository.IClient
	cfg         *RemoverConfig
}

func NewRemover(minioClient *minio.Client, grpcClient grpcRepository.IClient, cfg *RemoverConfig) *Remover {
	return &Remover{
		minioClient: minioClient,
		grpcClient:  grpcClient,
		cfg:         cfg,
	}
}

func (r *Remover) Remove(ctx context.Context, bucketName, objectName string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(r.cfg.Timeout)*time.Millisecond)
	defer cancel()

	err := r.minioClient.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		if _, logErr := r.grpcClient.AddLog(ctx, "failed to remove object", err.Error()); logErr != nil {
			logger.Error("can't send log to manager", "err", logErr)
		}

		return err
	}

	return nil
}

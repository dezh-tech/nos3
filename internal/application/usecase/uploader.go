package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"nos3/internal/domain/entity"
	"nos3/internal/domain/model"
	"nos3/internal/infrastructure/broker"
	"nos3/internal/infrastructure/database"
	"nos3/internal/infrastructure/minio"
	"nos3/pkg/logger"
)

type Uploader struct {
	publisher      *broker.Publisher
	writer         *database.BlobWriter
	minioUploader  *minio.Uploader
	minioRemover   *minio.Remover
	dbRemover      *database.BlobRemover
	defaultAddress string
}

func NewUploader(publisher *broker.Publisher, writer *database.BlobWriter,
	minioUploader *minio.Uploader, minioRemover *minio.Remover, dbRemover *database.BlobRemover, address string,
) *Uploader {
	return &Uploader{
		publisher:      publisher,
		writer:         writer,
		minioUploader:  minioUploader,
		minioRemover:   minioRemover,
		dbRemover:      dbRemover,
		defaultAddress: address,
	}
}

func (u *Uploader) Upload(ctx context.Context, body io.ReadCloser, fileSize int64,
	expectedHash, expectedType, author string,
) (entity.UploadResult, error) {
	result, err := u.minioUploader.UploadFile(ctx, body, fileSize, expectedHash, expectedType)
	if err != nil {
		logger.Error("failed to upload file to storage server", "err", err)

		return entity.UploadResult{}, errors.New("failed to upload file to storage server")
	}

	err = u.writer.Write(ctx, &model.Blob{
		ID:           expectedHash,
		Bucket:       result.Bucket,
		MinIOAddress: result.Location,
		UploadTime:   time.Now(),
		Author:       author,
		BlobType:     result.Type,
		Duration:     nil,
		Dimensions:   nil,
		Size:         result.Size,
		Blurhash:     "",
		Metadata:     nil,
	})
	if err != nil {
		if fileErr := u.minioRemover.Remove(ctx, result.Bucket, expectedHash); fileErr != nil {
			logger.Error("failed to remove file from minio after publish failed", "err", fileErr)
		}

		logger.Error("failed to add blob to db", "err", err)

		return entity.UploadResult{}, errors.New("failed to add blob to db")
	}

	err = u.publisher.Publish(ctx, expectedHash)
	if err != nil {
		if fileErr := u.minioRemover.Remove(ctx, result.Bucket, expectedHash); fileErr != nil {
			logger.Error("failed to remove file from minio after publish failed", "err", fileErr)
		}

		if removeErr := u.dbRemover.RemoveByHash(ctx, expectedHash); removeErr != nil {
			logger.Error("failed to remove blob from db after publish failed", "err", removeErr)
		}

		logger.Error("failed to publish message to broker for further processing", "err", err)

		return entity.UploadResult{}, errors.New("failed to publish message to broker for further processing")
	}

	return entity.UploadResult{
		Location: fmt.Sprintf("%s/%s", u.defaultAddress, expectedHash),
		Type:     result.Type,
		Size:     result.Size,
	}, nil
}

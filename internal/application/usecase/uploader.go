package usecase

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"nos3/internal/domain/entity"
	"nos3/internal/domain/model"
	"nos3/internal/domain/repository/broker"
	"nos3/internal/domain/repository/database"
	"nos3/internal/domain/repository/minio"
	"nos3/pkg/logger"
	"nos3/pkg/utils"
)

type Uploader struct {
	publisher      broker.Publisher
	writer         database.Writer
	retriever      database.Retriever
	minioUploader  minio.Uploader
	minioRemover   minio.Remover
	dbRemover      database.Remover
	defaultAddress string
}

func NewUploader(publisher broker.Publisher, retriever database.Retriever, writer database.Writer,
	minioUploader minio.Uploader, minioRemover minio.Remover, dbRemover database.Remover, address string,
) *Uploader {
	return &Uploader{
		publisher:      publisher,
		writer:         writer,
		retriever:      retriever,
		minioUploader:  minioUploader,
		minioRemover:   minioRemover,
		dbRemover:      dbRemover,
		defaultAddress: address,
	}
}

func (u *Uploader) Upload(ctx context.Context, body io.ReadCloser, fileSize int64,
	expectedHash, expectedType, author string,
) (entity.UploadResult, error) {
	_, err := u.retriever.GetByID(ctx, expectedHash)
	if err == nil {
		return entity.UploadResult{
			Location: "",
			Type:     "",
			Size:     0,
			Status:   http.StatusBadRequest,
		}, errors.New("a blob with the same hash already exists")
	}

	result, err := u.minioUploader.UploadFile(ctx, body, fileSize, expectedHash, expectedType)
	if err != nil {
		return entity.UploadResult{
			Status: result.HTTPStatus,
		}, err
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

		return entity.UploadResult{
			Status: http.StatusInternalServerError,
		}, errors.New("couldn't add blob to database")
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

		return entity.UploadResult{
			Status: http.StatusInternalServerError,
		}, errors.New("failed to publish blob to queue for further processing")
	}

	fileExtension := utils.GetExtensionFromMimeType(result.Type)

	return entity.UploadResult{
		Location: fmt.Sprintf("%s/%s%s", u.defaultAddress, expectedHash, fileExtension),
		Type:     result.Type,
		Size:     result.Size,
		Status:   http.StatusOK,
	}, nil
}

package minio

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"strings"
	"time"

	"nos3/internal/domain/entity"
	grpcRepository "nos3/internal/domain/repository/grpcclient"
	"nos3/pkg/logger"

	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

type Uploader struct {
	minioClient *minio.Client
	grpcClient  grpcRepository.IClient
	cfg         UploaderConfig
}

func NewUploader(minioClient *minio.Client, grpcClient grpcRepository.IClient, config UploaderConfig) *Uploader {
	return &Uploader{
		minioClient: minioClient,
		grpcClient:  grpcClient,
		cfg:         config,
	}
}

func (u *Uploader) UploadFile(ctx context.Context, body io.ReadCloser, fileSize int64,
	expectedHash, expectedType string,
) (entity.MinIOUploadResult, error) {
	defer body.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Duration(u.cfg.Timeout)*time.Millisecond)
	defer cancel()

	bucketName := u.cfg.Bucket
	var chunkNames []string
	hasher := sha256.New()

	detectedMIME, totalBytes, err := u.processFileChunks(ctx, body, bucketName, &chunkNames, hasher, expectedType)
	if err != nil {
		u.cleanupChunks(ctx, bucketName, chunkNames)

		return u.wrapErrorResult(http.StatusBadRequest, "invalid or unreadable file", err), err
	}

	if len(chunkNames) == 0 {
		return entity.MinIOUploadResult{
			HTTPStatus: http.StatusBadRequest,
		}, errors.New("read error: empty file")
	}

	if err := u.validateFileSize(totalBytes, fileSize); err != nil {
		u.cleanupChunks(ctx, bucketName, chunkNames)

		return u.wrapErrorResult(http.StatusBadRequest, "file size mismatch", err), err
	}

	calculatedHash := hex.EncodeToString(hasher.Sum(nil))
	if err := u.validateFileHash(calculatedHash, expectedHash); err != nil {
		u.cleanupChunks(ctx, bucketName, chunkNames)

		return u.wrapErrorResult(http.StatusBadRequest, "file hash mismatch", err), err
	}

	finalName := calculatedHash
	location, err := u.composeChunks(ctx, bucketName, chunkNames, finalName)
	u.cleanupChunks(ctx, bucketName, chunkNames)
	if err != nil {
		return u.wrapErrorResult(http.StatusInternalServerError, "failed to compose uploaded file", err), err
	}

	return entity.MinIOUploadResult{
		Size:       totalBytes,
		Type:       detectedMIME,
		Location:   location,
		Bucket:     bucketName,
		HTTPStatus: http.StatusOK,
	}, nil
}

func (u *Uploader) processFileChunks(ctx context.Context, body io.ReadCloser, bucketName string,
	chunkNames *[]string, hasher hash.Hash, expectedType string,
) (string, int64, error) {
	var detectedMIME string
	var totalBytes int64
	buf := make([]byte, 5*1024*1024)
	chunkIndex := 0

	for {
		n, err := body.Read(buf)
		if n > 0 { //nolint
			chunk := buf[:n]
			_, _ = hasher.Write(chunk)

			if chunkIndex == 0 {
				detectedMIME = mimetype.Detect(chunk).String()
				if !strings.Contains(detectedMIME, expectedType) {
					return "", 0, fmt.Errorf("invalid file type: detected %s, expected %s", detectedMIME, expectedType)
				}
			}

			chunkName := fmt.Sprintf("chunk-%s-%d", uuid.New().String(), chunkIndex)
			*chunkNames = append(*chunkNames, chunkName)

			_, err := u.minioClient.PutObject(ctx, bucketName, chunkName, bytes.NewReader(chunk), int64(len(chunk)),
				minio.PutObjectOptions{ContentType: detectedMIME})
			if err != nil {
				u.logInternalError(ctx, "failed to upload chunk", fmt.Sprintf("chunk: %s, error: %v", chunkName, err))

				return "", 0, fmt.Errorf("chunk upload failed: %w", err)
			}

			totalBytes += int64(len(chunk))
			chunkIndex++
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			u.logInternalError(ctx, "read error", err.Error())

			return "", 0, errors.New("failed to read file content")
		}
	}

	return detectedMIME, totalBytes, nil
}

func (u *Uploader) composeChunks(ctx context.Context, bucketName string, chunkNames []string,
	finalName string,
) (string, error) {
	sources := make([]minio.CopySrcOptions, len(chunkNames))
	for i, name := range chunkNames {
		sources[i] = minio.CopySrcOptions{Bucket: bucketName, Object: name}
	}
	dst := minio.CopyDestOptions{Bucket: bucketName, Object: finalName}
	info, err := u.minioClient.ComposeObject(ctx, dst, sources...)
	if err != nil {
		u.logInternalError(ctx, "failed to compose chunks", err.Error())

		return "", errors.New("compose operation failed")
	}

	return info.Location, nil
}

func (u *Uploader) validateFileSize(totalBytes, expectedSize int64) error {
	if totalBytes != expectedSize && expectedSize != -1 {
		return fmt.Errorf("file size mismatch: read %d bytes, expected %d", totalBytes, expectedSize)
	}

	return nil
}

func (u *Uploader) validateFileHash(calculatedHash, givenHash string) error {
	if calculatedHash != givenHash {
		return fmt.Errorf("invalid hash: got %s, expected %s", givenHash, calculatedHash)
	}

	return nil
}

func (u *Uploader) cleanupChunks(ctx context.Context, bucketName string, chunkNames []string) {
	for _, name := range chunkNames {
		err := u.minioClient.RemoveObject(ctx, bucketName, name, minio.RemoveObjectOptions{})
		if err != nil {
			u.logInternalError(ctx, "failed to cleanup chunk", err.Error())
		}
	}
}

func (u *Uploader) logInternalError(ctx context.Context, title, detail string) {
	if _, err := u.grpcClient.AddLog(ctx, title, detail); err != nil {
		logger.Error("can't send log to manager", "err", err)
	}
}

func (u *Uploader) wrapErrorResult(status int, message string, err error) entity.MinIOUploadResult {
	logger.Error(message, "err", err)

	return entity.MinIOUploadResult{
		HTTPStatus: status,
	}
}

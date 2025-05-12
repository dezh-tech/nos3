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
	"strings"
	"time"

	"github.com/dezh-tech/immortal/pkg/logger"

	grpcRepository "nos3/internal/domain/repository/grpcclient"

	"github.com/gabriel-vasile/mimetype"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

type Uploader struct {
	minioClient *minio.Client
	grpcClient  grpcRepository.IClient
	cfg         *UploaderConfig
}

func NewUploader(minioClient *minio.Client, grpcClient grpcRepository.IClient, config *UploaderConfig) *Uploader {
	return &Uploader{
		minioClient: minioClient,
		grpcClient:  grpcClient,
		cfg:         config,
	}
}

type UploadFileResult struct {
	Size int64  `json:"size"`
	Type string `json:"type"`
}

func (u *Uploader) UploadFile(ctx context.Context, body io.ReadCloser, fileSize int64, expectedHash,
	expectedType string,
) (UploadFileResult, error) {
	defer body.Close()

	ctx, cancel := context.WithTimeout(ctx, time.Duration(u.cfg.Timeout)*time.Millisecond)
	defer cancel()

	bucketName := u.cfg.Bucket
	var chunkNames []string
	hasher := sha256.New()

	detectedMIME, totalBytes, err := u.processFileChunks(ctx, body, bucketName, &chunkNames, hasher, expectedType)
	if err != nil {
		u.cleanupChunks(ctx, bucketName, chunkNames)

		return UploadFileResult{}, err
	}

	if len(chunkNames) == 0 {
		return UploadFileResult{}, errors.New("read error: empty file")
	}

	if err := u.validateFileSize(totalBytes, fileSize); err != nil {
		u.cleanupChunks(ctx, bucketName, chunkNames)

		return UploadFileResult{}, err
	}

	calculatedHash := hex.EncodeToString(hasher.Sum(nil))
	if err := u.validateFileHash(calculatedHash, expectedHash); err != nil {
		u.cleanupChunks(ctx, bucketName, chunkNames)

		return UploadFileResult{}, err
	}

	finalName := calculatedHash
	if err := u.composeChunks(ctx, bucketName, chunkNames, finalName); err != nil {
		u.cleanupChunks(ctx, bucketName, chunkNames)

		return UploadFileResult{}, err
	}

	u.cleanupChunks(ctx, bucketName, chunkNames)

	return UploadFileResult{
		Size: totalBytes,
		Type: detectedMIME,
	}, nil
}

func (u *Uploader) processFileChunks(ctx context.Context, body io.ReadCloser, bucketName string, chunkNames *[]string,
	hasher hash.Hash, expectedType string,
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
				minio.PutObjectOptions{
					ContentType: detectedMIME,
				})
			if err != nil {
				if _, logErr := u.grpcClient.AddLog(ctx, "failed to upload chunk",
					fmt.Sprintf("chunk: %s, error: %v", chunkName, err)); logErr != nil {
					logger.Error("can't send log to manager", "err", logErr)
				}

				return "", 0, fmt.Errorf("chunk upload failed: %w", err)
			}

			totalBytes += int64(len(chunk))
			chunkIndex++
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.Error("read error", "err", err.Error())

			return "", 0, fmt.Errorf("read error: %w", err)
		}
	}

	return detectedMIME, totalBytes, nil
}

func (u *Uploader) composeChunks(ctx context.Context, bucketName string, chunkNames []string, finalName string) error {
	sources := make([]minio.CopySrcOptions, len(chunkNames))
	for i, name := range chunkNames {
		sources[i] = minio.CopySrcOptions{Bucket: bucketName, Object: name}
	}

	dst := minio.CopyDestOptions{Bucket: bucketName, Object: finalName}
	_, err := u.minioClient.ComposeObject(ctx, dst, sources...)
	if err != nil {
		if _, logErr := u.grpcClient.AddLog(ctx, "failed to compose chunks", err.Error()); logErr != nil {
			logger.Error("can't send log to manager", "err", logErr)
		}

		return fmt.Errorf("compose error: %w", err)
	}

	return nil
}

func (u *Uploader) validateFileSize(totalBytes, expectedSize int64) error {
	if totalBytes != expectedSize && expectedSize != -1 {
		return fmt.Errorf("file size mismatch: read %d bytes, expected %d", totalBytes, expectedSize)
	}

	return nil
}

func (u *Uploader) validateFileHash(calculatedHash, expectedHash string) error {
	if calculatedHash != expectedHash {
		return fmt.Errorf("invalid hash: got %s, expected %s", calculatedHash, expectedHash)
	}

	return nil
}

func (u *Uploader) cleanupChunks(ctx context.Context, bucketName string, chunkNames []string) {
	for _, name := range chunkNames {
		err := u.minioClient.RemoveObject(ctx, bucketName, name, minio.RemoveObjectOptions{})
		if err != nil {
			if _, logErr := u.grpcClient.AddLog(ctx, "failed to cleanup chunk", err.Error()); logErr != nil {
				logger.Error("can't send log to manager", "err", logErr)
			}
		}
	}
}

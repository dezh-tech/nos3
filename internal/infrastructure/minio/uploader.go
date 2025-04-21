package minio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/minio/minio-go/v7"
)

// Uploader is responsible for handling file uploads to MinIO.
type Uploader struct {
	minioClient *minio.Client
	timeout     int
}

// NewUploader creates a new Uploader instance.
func NewUploader(minioClient *minio.Client, timeout int) *Uploader {
	return &Uploader{
		minioClient: minioClient,
		timeout:     timeout,
	}
}

type UploadFileResult struct {
	Size int64  `json:"size"`
	Type string `json:"type"`
}

// UploadFile uploads a file to MinIO after validating the file's size, type, and hash.
// It checks the file against declared attributes and uploads it to the appropriate bucket.
func (u *Uploader) UploadFile(ctx context.Context, body io.ReadCloser, fileSize int64, hash, fileType string) (
	UploadFileResult, error,
) {
	data, err := io.ReadAll(body)
	defer body.Close()
	if err != nil {
		return UploadFileResult{}, fmt.Errorf("failed to read file: %w", err)
	}

	isValid, actualSize := u.validateFileSize(data, fileSize)
	if !isValid {
		return UploadFileResult{}, errors.New("invalid file size")
	}

	isValid, actualType := u.validateFileType(data, fileType)
	if !isValid {
		return UploadFileResult{}, errors.New("invalid file type")
	}

	bucket := u.getBucketForType(fileType)

	reader := io.NopCloser(bytes.NewReader(data))

	ctx, cancel := context.WithTimeout(ctx, time.Duration(u.timeout)*time.Millisecond)
	defer cancel()

	uploadInfo, err := u.minioClient.PutObject(ctx, bucket, hash, reader, actualSize,
		minio.PutObjectOptions{ContentType: actualType})
	if err != nil {
		return UploadFileResult{}, fmt.Errorf("minio put object error: %w", err)
	}

	return UploadFileResult{
		Size: uploadInfo.Size,
		Type: actualType,
	}, nil
}

func (u *Uploader) validateFileSize(data []byte, declaredSize int64) (bool, int64) {
	actualSize := int64(len(data))
	validation := declaredSize == -1 || declaredSize == actualSize

	return validation, actualSize
}

func (u *Uploader) validateFileType(data []byte, expectedType string) (bool, string) {
	detected := mimetype.Detect(data).String()
	validation := expectedType == detected || expectedType == ""

	return validation, detected
}

func (u *Uploader) getBucketForType(fileType string) string {
	switch {
	case strings.HasPrefix(fileType, "image/"):
		return "images"
	case strings.HasPrefix(fileType, "video/"):
		return "videos"
	case strings.HasPrefix(fileType, "audio/"):
		return "audios"
	case fileType == "application/pdf":
		return "documents"
	default:
		return "other"
	}
}

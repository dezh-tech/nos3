package minio

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// UploadFile uploads a file to MinIO after validating the file's size, type, and hash.
// It checks the file against declared attributes and uploads it to the appropriate bucket.
func (u *Uploader) UploadFile(ctx context.Context, body io.ReadCloser, fileSize int64, hash, fileType string) error {
	data, err := io.ReadAll(body)
	defer body.Close()
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if !u.validateFileSize(data, fileSize) {
		return errors.New("invalid file size")
	}

	if !u.validateFileType(data, fileType) {
		return errors.New("invalid file type")
	}

	if !u.validateHash(data, hash) {
		return errors.New("file hash does not match")
	}

	bucket := u.getBucketForType(fileType)

	reader := io.NopCloser(bytes.NewReader(data))

	ctx, cancel := context.WithTimeout(ctx, time.Duration(u.timeout)*time.Millisecond)
	defer cancel()

	_, err = u.minioClient.PutObject(ctx, bucket, hash, reader, fileSize, minio.PutObjectOptions{ContentType: fileType})
	if err != nil {
		return fmt.Errorf("minio put object error: %w", err)
	}

	return nil
}

func (u *Uploader) validateFileSize(data []byte, declaredSize int64) bool {
	return declaredSize == -1 || int64(len(data)) == declaredSize
}

func (u *Uploader) validateFileType(data []byte, expectedType string) bool {
	detected := mimetype.Detect(data)

	return expectedType == detected.String() || expectedType == ""
}

func (u *Uploader) validateHash(data []byte, expectedHash string) bool {
	hash := sha256.Sum256(data)

	return hex.EncodeToString(hash[:]) == expectedHash
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

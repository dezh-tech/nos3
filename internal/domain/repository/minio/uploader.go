package minio

import (
	"context"
	"io"
)

type Uploader interface {
	UploadFile(ctx context.Context, body io.ReadCloser, fileSize int64, hash, fileType string) error
}

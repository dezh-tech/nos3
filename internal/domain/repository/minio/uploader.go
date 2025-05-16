package minio

import (
	"context"
	"io"

	"nos3/internal/domain/entity"
)

type Uploader interface {
	UploadFile(ctx context.Context, body io.ReadCloser, fileSize int64, expectedHash,
		expectedType string,
	) (entity.UploadResult, error)
}

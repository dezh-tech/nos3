package abstraction

import (
	"context"
	"io"

	"nos3/internal/domain/entity"
)

type Uploader interface {
	Upload(ctx context.Context, body io.ReadCloser, fileSize int64,
		expectedHash, expectedType, author string) (entity.UploadResult, error)
}

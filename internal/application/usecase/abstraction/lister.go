package abstraction

import (
	"context"
	"time"

	"nos3/internal/domain/dto"
)

type Lister interface {
	ListBlobs(ctx context.Context, pubKey string, since, until *time.Time) ([]dto.BlobDescriptor,
		int, error)
}

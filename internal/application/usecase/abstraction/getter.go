package abstraction

import (
	"context"

	"nos3/internal/domain/model"
)

// Getter defines the interface for retrieving blob information.
type Getter interface {
	GetBlob(ctx context.Context, hash string) (*model.Blob, error)
}

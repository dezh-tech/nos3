package database

import (
	"context"
	"time"

	"nos3/internal/domain/model"
)

// Lister defines the interface for listing blobs from the database.
type Lister interface {
	GetByAuthor(ctx context.Context, author string, since, until *time.Time) ([]model.Blob, error)
}

package database

import (
	"context"

	"nos3/internal/domain/model"
)

type Writer interface {
	Write(ctx context.Context, blob *model.Blob) error
}

package database

import (
	"context"

	"nos3/internal/domain/model"
)

type Writer interface {
	Upload(ctx context.Context, media *model.Media) error
}

package database

import (
	"context"

	"nos3/internal/domain/model"
)

type Retriever interface {
	GetByID(ctx context.Context, id string) (*model.Blob, error)
}

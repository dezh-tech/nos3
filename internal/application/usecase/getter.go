package usecase

import (
	"context"
	"errors"

	"nos3/internal/domain/model"
	"nos3/internal/domain/repository/database"
)

// Getter implements the Getter abstraction for retrieving blob information.
type Getter struct {
	retriever database.Retriever
}

// NewGetter creates a new Getter usecase.
func NewGetter(retriever database.Retriever) *Getter {
	return &Getter{
		retriever: retriever,
	}
}

// GetBlob retrieves a blob by its SHA256 hash from the database.
func (g *Getter) GetBlob(ctx context.Context, hash string) (*model.Blob, error) {
	blob, err := g.retriever.GetByID(ctx, hash)
	if err != nil {
		return nil, errors.New("blob not found")
	}

	return blob, nil
}

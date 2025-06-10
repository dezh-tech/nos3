package usecase

import (
	"context"
	"errors"
	"net/http"

	"nos3/internal/domain/repository/database"
	"nos3/internal/domain/repository/minio"
)

// Deleter implements the Deleter abstraction for deleting blob information.
type Deleter struct {
	dbRetriever  database.Retriever
	dbRemover    database.Remover
	minioRemover minio.Remover
}

// NewDeleter creates a new Deleter usecase.
func NewDeleter(dbRetriever database.Retriever, dbRemover database.Remover, minioRemover minio.Remover) *Deleter {
	return &Deleter{
		dbRetriever:  dbRetriever,
		dbRemover:    dbRemover,
		minioRemover: minioRemover,
	}
}

// DeleteBlob deletes a blob by its SHA256 hash from MinIO and the database.
func (d *Deleter) DeleteBlob(ctx context.Context, sha256 string) (int, error) {
	blob, err := d.dbRetriever.GetByID(ctx, sha256)
	if err != nil {
		return http.StatusNotFound, errors.New("blob not found")
	}

	if err := d.minioRemover.Remove(ctx, blob.Bucket, blob.ID); err != nil {
		return http.StatusInternalServerError, errors.New("failed to remove blob from storage")
	}

	if err := d.dbRemover.RemoveByHash(ctx, sha256); err != nil {
		return http.StatusInternalServerError, errors.New("failed to remove blob from database")
	}

	return http.StatusOK, nil
}

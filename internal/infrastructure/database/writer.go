package database

import (
	"context"

	"nos3/internal/domain/model"
)

type BlobWriter struct {
	db *Database
}

func NewBlobWriter(db *Database) *BlobWriter {
	return &BlobWriter{db: db}
}

func (u *BlobWriter) Write(ctx context.Context, blob *model.Blob) error {
	ctx, cancel := context.WithTimeout(ctx, u.db.QueryTimeout)
	defer cancel()

	coll := u.db.Client.Database(u.db.DBName).Collection(BlobCollection)

	_, err := coll.InsertOne(ctx, blob)

	return err
}

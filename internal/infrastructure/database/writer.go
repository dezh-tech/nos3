package database

import (
	"context"

	"nos3/internal/domain/model"
)

type MediaWriter struct {
	db *Database
}

func NewMediaWriter(db *Database) *MediaWriter {
	return &MediaWriter{db: db}
}

func (u *MediaWriter) Write(ctx context.Context, media *model.Media) error {
	ctx, cancel := context.WithTimeout(ctx, u.db.QueryTimeout)
	defer cancel()

	coll := u.db.Client.Database(u.db.DBName).Collection(MediaCollection)

	_, err := coll.InsertOne(ctx, media)

	return err
}

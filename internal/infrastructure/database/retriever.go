package database

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"

	"nos3/internal/domain/model"
)

type MediaRetriever struct {
	db *Database
}

func NewMediaRetriever(db *Database) *MediaRetriever {
	return &MediaRetriever{db: db}
}

func (r *MediaRetriever) GetByID(ctx context.Context, id string) (*model.Media, error) {
	ctx, cancel := context.WithTimeout(ctx, r.db.QueryTimeout)
	defer cancel()

	coll := r.db.Client.Database(r.db.DBName).Collection(MediaCollection)

	var media model.Media
	err := coll.FindOne(ctx, bson.M{"_id": id}).Decode(&media)
	if err != nil {
		return nil, err
	}

	return &media, nil
}

package database

import (
	"context"

	"nos3/internal/domain/model"

	"go.mongodb.org/mongo-driver/bson"
)

type BlobRetriever struct {
	db *Database
}

func NewBlobRetriever(db *Database) *BlobRetriever {
	return &BlobRetriever{db: db}
}

func (r *BlobRetriever) GetByID(ctx context.Context, id string) (*model.Blob, error) {
	ctx, cancel := context.WithTimeout(ctx, r.db.QueryTimeout)
	defer cancel()

	coll := r.db.Client.Database(r.db.DBName).Collection(BlobCollection)

	var blob model.Blob
	err := coll.FindOne(ctx, bson.M{"_id": id}).Decode(&blob)
	if err != nil {
		return nil, err
	}

	return &blob, nil
}

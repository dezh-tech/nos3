package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"nos3/internal/domain/model"
	grpcRepository "nos3/internal/domain/repository/grpcclient"
	"nos3/pkg/logger"
)

type BlobLister struct {
	db         *Database
	grpcClient grpcRepository.IClient
}

func NewBlobLister(db *Database, grpcClient grpcRepository.IClient) *BlobLister {
	return &BlobLister{
		db:         db,
		grpcClient: grpcClient,
	}
}

func (l *BlobLister) GetByAuthor(ctx context.Context, author string, since, until *time.Time) ([]model.Blob, error) {
	ctx, cancel := context.WithTimeout(ctx, l.db.QueryTimeout)
	defer cancel()

	coll := l.db.Client.Database(l.db.DBName).Collection(BlobCollection)

	filter := bson.M{"author": author}

	if since != nil || until != nil {
		uploadedFilter := bson.M{}
		if since != nil {
			uploadedFilter["$gte"] = *since
		}
		if until != nil {
			uploadedFilter["$lte"] = *until
		}
		filter["upload_time"] = uploadedFilter
	}

	cursor, err := coll.Find(ctx, filter)
	if err != nil {
		if _, logErr := l.grpcClient.AddLog(ctx, "failed to retrieve blobs by author", err.Error()); logErr != nil {
			logger.Error("can't send log to manager", "err", logErr)
		}

		return nil, err
	}
	defer cursor.Close(ctx)

	var blobs []model.Blob
	if err = cursor.All(ctx, &blobs); err != nil {
		if _, logErr := l.grpcClient.AddLog(ctx, "failed to decode blobs", err.Error()); logErr != nil {
			logger.Error("can't send log to manager", "err", logErr)
		}

		return nil, err
	}

	return blobs, nil
}

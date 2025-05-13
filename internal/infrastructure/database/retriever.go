package database

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"

	"nos3/internal/domain/model"
	grpcRepository "nos3/internal/domain/repository/grpcclient"
	"nos3/pkg/logger"
)

type BlobRetriever struct {
	db         *Database
	grpcClient grpcRepository.IClient
}

func NewBlobRetriever(db *Database, grpcClient grpcRepository.IClient) *BlobRetriever {
	return &BlobRetriever{
		db:         db,
		grpcClient: grpcClient,
	}
}

func (r *BlobRetriever) GetByID(ctx context.Context, id string) (*model.Blob, error) {
	ctx, cancel := context.WithTimeout(ctx, r.db.QueryTimeout)
	defer cancel()

	coll := r.db.Client.Database(r.db.DBName).Collection(BlobCollection)

	var blob model.Blob
	err := coll.FindOne(ctx, bson.M{"_id": id}).Decode(&blob)
	if err != nil {
		if _, logErr := r.grpcClient.AddLog(ctx, "failed to retrieve blob by id", err.Error()); logErr != nil {
			logger.Error("can't send log to manager", "err", logErr)
		}

		return nil, err
	}

	return &blob, nil
}

package database

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	grpcRepository "nos3/internal/domain/repository/grpcclient"
	"nos3/pkg/logger"
)

type BlobRemover struct {
	db         *Database
	grpcClient grpcRepository.IClient
}

func NewRemover(db *Database, grpcClient grpcRepository.IClient) *BlobRemover {
	return &BlobRemover{
		db:         db,
		grpcClient: grpcClient,
	}
}

func (r *BlobRemover) RemoveByHash(ctx context.Context, hash string) error {
	ctx, cancel := context.WithTimeout(ctx, r.db.QueryTimeout)
	defer cancel()

	coll := r.db.Client.Database(r.db.DBName).Collection(BlobCollection)
	_, err := coll.DeleteOne(ctx, bson.M{"hash": hash}, &options.DeleteOptions{})
	if err != nil {
		if _, logErr := r.grpcClient.AddLog(ctx, "failed to remove blob", err.Error()); logErr != nil {
			logger.Error("can't send log to manager", "err", logErr)
		}

		return err
	}

	return nil
}

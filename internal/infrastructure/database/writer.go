package database

import (
	"context"

	"nos3/pkg/logger"

	"nos3/internal/domain/model"
	grpcRepository "nos3/internal/domain/repository/grpcclient"
)

type BlobWriter struct {
	db         *Database
	grpcClient grpcRepository.IClient
}

func NewBlobWriter(db *Database, grpcClient grpcRepository.IClient) *BlobWriter {
	return &BlobWriter{
		db:         db,
		grpcClient: grpcClient,
	}
}

func (u *BlobWriter) Write(ctx context.Context, blob *model.Blob) error {
	ctx, cancel := context.WithTimeout(ctx, u.db.QueryTimeout)
	defer cancel()

	coll := u.db.Client.Database(u.db.DBName).Collection(BlobCollection)

	_, err := coll.InsertOne(ctx, blob)
	if err != nil {
		if _, logErr := u.grpcClient.AddLog(ctx, "failed to write blob to database", err.Error()); logErr != nil {
			logger.Error("can't send log to manager", "err", logErr)
		}

		return err
	}

	return nil
}

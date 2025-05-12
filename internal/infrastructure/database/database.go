package database

import (
	"context"
	"time"

	grpcRepository "nos3/internal/domain/repository/grpcclient"

	"github.com/dezh-tech/immortal/pkg/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const BlobCollection = "blob"

type Database struct {
	DBName       string
	QueryTimeout time.Duration
	Client       *mongo.Client
	grpcClient   grpcRepository.IClient
}

func Connect(cfg Config, grpcClient grpcRepository.IClient) (*Database, error) {
	logger.Info("connecting to database")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ConnectionTimeout)*time.Millisecond)
	defer cancel()

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(cfg.URI).
		SetServerAPIOptions(serverAPI).
		SetConnectTimeout(time.Duration(cfg.ConnectionTimeout) * time.Millisecond).
		SetBSONOptions(&options.BSONOptions{
			UseJSONStructTags: true,
			NilSliceAsEmpty:   true,
		})

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		if _, logErr := grpcClient.AddLog(ctx, "failed to connect to MongoDB", err.Error()); logErr != nil {
			logger.Error("can't send log to manager", "err", logErr)
		}

		return nil, err
	}

	qCtx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.QueryTimeout)*time.Millisecond)
	defer cancel()

	if err := client.Ping(qCtx, nil); err != nil {
		if _, logErr := grpcClient.AddLog(ctx, "failed to ping MongoDB", err.Error()); logErr != nil {
			logger.Error("can't send log to manager", "err", logErr)
		}

		return nil, err
	}

	db := &Database{
		Client:       client,
		DBName:       cfg.DBName,
		QueryTimeout: time.Duration(cfg.QueryTimeout) * time.Millisecond,
		grpcClient:   grpcClient,
	}

	if err := initBlobCollection(db); err != nil {
		logger.Error("failed to initialize blob collection", err.Error())

		return nil, err
	}

	return db, nil
}

func initBlobCollection(db *Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), db.QueryTimeout)
	defer cancel()

	collections, err := db.Client.Database(db.DBName).ListCollectionNames(ctx, bson.M{"name": BlobCollection})
	if err != nil {
		if _, logErr := db.grpcClient.AddLog(ctx, "failed to list collection names", err.Error()); logErr != nil {
			logger.Error("can't send log to manager", "err", logErr)
		}

		return err
	}
	if len(collections) > 0 {
		return nil // already exists
	}

	collOpts := options.CreateCollection().SetValidator(bson.M{
		"$jsonSchema": bson.M{
			"bsonType": "object",
			"required": []string{"_id", "minio_address", "upload_time", "author", "blob_type"},
			"properties": bson.M{
				"_id": bson.M{
					"bsonType":    "string",
					"minLength":   64,
					"maxLength":   64,
					"description": "must be 64-character SHA hash",
				},
				"minio_address": bson.M{"bsonType": "string"},
				"upload_time":   bson.M{"bsonType": "date"},
				"author": bson.M{
					"bsonType": "string",
					"pattern":  "^[a-fA-F0-9]{64}$",
				},
				"blob_type": bson.M{"bsonType": "string"},
				"duration":  bson.M{"bsonType": []string{"int", "null"}},
				"dimensions": bson.M{
					"bsonType": []string{"object", "null"},
					"properties": bson.M{
						"width":  bson.M{"bsonType": "int"},
						"height": bson.M{"bsonType": "int"},
					},
				},
				"size":     bson.M{"bsonType": "long"},
				"blurhash": bson.M{"bsonType": "string"},
				"metadata": bson.M{
					"bsonType": "array",
					"items": bson.M{
						"bsonType": "object",
						"required": []string{"key", "value"},
						"properties": bson.M{
							"key":   bson.M{"bsonType": "string"},
							"value": bson.M{"bsonType": "string"},
						},
					},
				},
			},
		},
	})

	err = db.Client.Database(db.DBName).CreateCollection(ctx, BlobCollection, collOpts)
	if err != nil {
		if _, logErr := db.grpcClient.AddLog(ctx, "failed to create collection", err.Error()); logErr != nil {
			logger.Error("can't send log to manager", "err", logErr)
		}

		return err
	}

	coll := db.Client.Database(db.DBName).Collection(BlobCollection)
	_, err = coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "author", Value: 1}},
	})
	if err != nil {
		if _, logErr := db.grpcClient.AddLog(ctx, "failed to create index", err.Error()); logErr != nil {
			logger.Error("can't send log to manager", "err", logErr)
		}
	}

	return err
}

func (db *Database) Stop() error {
	if err := db.Client.Disconnect(context.Background()); err != nil {
		if _, logErr := db.grpcClient.AddLog(context.Background(),
			"failed to disconnect from MongoDB", err.Error()); logErr != nil {
			logger.Error("can't send log to manager", "err", logErr)
		}

		return err
	}

	return nil
}

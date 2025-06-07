package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"nos3/pkg/logger"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"

	"nos3/internal/application/usecase"
	"nos3/internal/domain/model"
	"nos3/internal/infrastructure/database"
	"nos3/internal/infrastructure/grpcclient"
	minioInfra "nos3/internal/infrastructure/minio"
	"nos3/internal/presentation"
	"nos3/internal/presentation/middleware"
)

func TestHandleDelete_Integration(t *testing.T) {
	services := setupServices(t)
	defer cleanupServices(t, services)

	addr, cleanup := startTestGRPCServer(t)
	defer cleanup()
	grpcClient, err := grpcclient.New(grpcclient.ClientConfig{
		Endpoint:  addr,
		Heartbeat: 30,
	})
	require.NoError(t, err)

	mongoEndpoint, err := services.mongoC.Endpoint(context.Background(), "")
	require.NoError(t, err)

	db, err := database.Connect(database.Config{
		URI:               fmt.Sprintf("mongodb://%s:%s@%s", mongoUser, mongoPassword, mongoEndpoint),
		DBName:            mongoDBName,
		ConnectionTimeout: 30000,
		QueryTimeout:      30000,
	}, grpcClient)
	require.NoError(t, err)
	defer func(db *database.Database) {
		err := db.Stop()
		if err != nil {
			logger.Error("couldn't stop db instance")
		}
	}(db)

	minioEndpoint, err := services.minioC.Endpoint(context.Background(), "")
	require.NoError(t, err)

	minioClient, err := minio.New(minioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioUser, minioPassword, ""),
		Secure: false,
	})
	require.NoError(t, err)

	dbRetriever := database.NewBlobRetriever(db, grpcClient)
	dbRemover := database.NewRemover(db, grpcClient)
	minioRemover := minioInfra.NewRemover(minioClient, grpcClient, minioInfra.RemoverConfig{Timeout: 3000})

	deleteHandler := NewDeleteHandler(usecase.NewDeleter(dbRetriever, dbRemover, minioRemover))

	e := echo.New()
	e.Use(echoMiddleware.Logger())
	e.Use(echoMiddleware.CORSWithConfig(echoMiddleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderAuthorization, echo.HeaderContentType, echo.HeaderContentLength},
		AllowMethods: []string{
			http.MethodGet, http.MethodPut, http.MethodPost,
			http.MethodDelete, http.MethodHead, http.MethodOptions,
		},
		MaxAge: 86400,
	}))
	e.Use(echoMiddleware.Recover())
	e.Use(echoMiddleware.Secure())
	e.Use(echoMiddleware.BodyLimit("50M"))
	e.Use(echoMiddleware.RateLimiter(echoMiddleware.NewRateLimiterMemoryStore(20)))

	e.DELETE(fmt.Sprintf("/:%s", presentation.Sha256Param), deleteHandler.HandleDelete,
		middleware.AuthMiddleware("delete"), middleware.AuthDeleteMiddleware())

	testCases := []struct {
		name            string
		setup           func(t *testing.T, minioClient *minio.Client, db *database.Database) (actualSha256 string, requestPath string, authHeader string)
		expectedStatus  int
		checkPostDelete func(t *testing.T, minioClient *minio.Client, db *database.Database, hash string)
		expectedReason  string
	}{
		{
			name: "Successful deletion of blob",
			setup: func(t *testing.T, minioClient *minio.Client, db *database.Database) (string, string, string) {
				t.Helper()
				content := []byte("content to delete")
				hashBytes := sha256.Sum256(content)
				hexHash := hex.EncodeToString(hashBytes[:])
				blobType := "text/plain"

				_, err := minioClient.PutObject(context.Background(),
					minioBucket, hexHash, bytes.NewReader(content), int64(len(content)),
					minio.PutObjectOptions{ContentType: blobType})
				require.NoError(t, err)

				blob := &model.Blob{
					ID:           hexHash,
					Bucket:       minioBucket,
					MinIOAddress: fmt.Sprintf("http://%s/%s/%s", minioEndpoint, minioBucket, hexHash),
					UploadTime:   time.Now(),
					Author:       "f2f357f4955a5b51b329381c828d5789f28d88e6e582d921532c25672c833d7b",
					BlobType:     blobType,
					Size:         int64(len(content)),
				}

				coll := db.Client.Database(db.DBName).Collection(database.BlobCollection)
				_, err = coll.InsertOne(context.Background(), blob)
				require.NoError(t, err)

				authHeader := generateValidAuthHeader(t, 600, "delete", hexHash, "")

				return hexHash, hexHash, authHeader
			},
			expectedStatus: http.StatusOK,
			checkPostDelete: func(t *testing.T, minioClient *minio.Client, db *database.Database, hash string) {
				t.Helper()

				time.Sleep(500 * time.Millisecond) // So delete operations take effect

				_, err := minioClient.StatObject(context.Background(), minioBucket, hash, minio.StatObjectOptions{})
				assert.Error(t, err, "Expected object to be deleted from MinIO")

				coll := db.Client.Database(db.DBName).Collection(database.BlobCollection)
				var blob model.Blob
				err = coll.FindOne(context.Background(), bson.M{"_id": hash}).Decode(&blob)
				assert.Error(t, err, "Expected blob to be deleted from database")
			},
		},
		{
			name: "Attempt to delete non-existent blob",
			setup: func(t *testing.T, _ *minio.Client, _ *database.Database) (string, string, string) {
				t.Helper()
				nonExistentHash := "nonexistenthash00000000000000000000000000000000000000000000000000000000"
				authHeader := generateValidAuthHeader(t, 600, "delete", nonExistentHash, "")

				return nonExistentHash, nonExistentHash, authHeader
			},
			expectedStatus: http.StatusNotFound,
			expectedReason: "blob not found",
		},
		{
			name: "Unauthorized: wrong action in Authorization header",
			setup: func(t *testing.T, _ *minio.Client, _ *database.Database) (string, string, string) {
				t.Helper()
				hash := "wrongactionhash00000000000000000000000000000000000000000000000000"

				return hash, hash, generateValidAuthHeader(t, 600, "upload", hash, "")
			},
			expectedStatus: http.StatusUnauthorized,
			expectedReason: "invalid action",
		},
		{
			name: "Unauthorized: x tag mismatch",
			setup: func(t *testing.T, _ *minio.Client, _ *database.Database) (string, string, string) {
				t.Helper()
				paramHash := "mismatchhash000000000000000000000000000000000000000000000000000000"
				authHash := "anotherhash0000000000000000000000000000000000000000000000000000000"

				return paramHash, paramHash, generateValidAuthHeader(t, 600, "delete", authHash, "")
			},
			expectedStatus: http.StatusUnauthorized,
			expectedReason: "x tag mismatch with URL sha256 for delete action",
		},
		{
			name: "Hash with extension for delete",
			setup: func(t *testing.T, minioClient *minio.Client, db *database.Database) (string, string, string) {
				t.Helper()
				content := []byte("content with extension for delete")
				hashBytes := sha256.Sum256(content)
				hexHash := hex.EncodeToString(hashBytes[:])
				blobType := "application/octet-stream"

				_, err := minioClient.PutObject(context.Background(),
					minioBucket, hexHash, bytes.NewReader(content), int64(len(content)),
					minio.PutObjectOptions{ContentType: blobType})
				require.NoError(t, err)

				blob := &model.Blob{
					ID:           hexHash,
					Bucket:       minioBucket,
					MinIOAddress: fmt.Sprintf("http://%s/%s/%s", minioEndpoint, minioBucket, hexHash),
					UploadTime:   time.Now(),
					Author:       "f2f357f4955a5b51b329381c828d5789f28d88e6e582d921532c25672c833d7b",
					BlobType:     blobType,
					Size:         int64(len(content)),
				}

				coll := db.Client.Database(db.DBName).Collection(database.BlobCollection)
				_, err = coll.InsertOne(context.Background(), blob)
				require.NoError(t, err)

				authHeader := generateValidAuthHeader(t, 600, "delete", hexHash, "")

				return hexHash, hexHash + ".bin", authHeader
			},
			expectedStatus: http.StatusOK,
			checkPostDelete: func(t *testing.T, minioClient *minio.Client, db *database.Database, hash string) {
				t.Helper()
				_, err := minioClient.StatObject(context.Background(), minioBucket, hash, minio.StatObjectOptions{})
				assert.Error(t, err, "Expected object to be deleted from MinIO")

				coll := db.Client.Database(db.DBName).Collection(database.BlobCollection)
				var blob model.Blob
				err = coll.FindOne(context.Background(), bson.M{"_id": hash}).Decode(&blob)
				assert.Error(t, err, "Expected blob to be deleted from database")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var actualSha256 string
			var requestPath string
			var authHeader string

			if tc.setup != nil {
				actualSha256, requestPath, authHeader = tc.setup(t, minioClient, db)
			} else {
				actualSha256 = "dummyhash00000000000000000000000000000000000000000000000000000000"
				requestPath = "dummyhash00000000000000000000000000000000000000000000000000000000"
				authHeader = generateValidAuthHeader(t, 600, "delete", actualSha256, "")
			}

			req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/%s", requestPath), http.NoBody)
			if authHeader != "" {
				req.Header.Set(presentation.AuthKey, authHeader)
			}

			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, tc.expectedStatus, rec.Code)
			if tc.expectedStatus == http.StatusOK {
				if tc.checkPostDelete != nil {
					tc.checkPostDelete(t, minioClient, db, actualSha256)
				}
			} else {
				assert.Contains(t, rec.Header().Get(presentation.ReasonTag), tc.expectedReason)
			}
		})
	}
}

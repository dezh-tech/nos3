package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"nos3/internal/application/usecase"
	"nos3/internal/domain/model"
	"nos3/internal/infrastructure/database"
	"nos3/internal/infrastructure/grpcclient"
	"nos3/internal/presentation"
	"nos3/internal/presentation/middleware"
)

func TestHandleGet_Integration(t *testing.T) {
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

	retriever := database.NewBlobRetriever(db, grpcClient)
	minioEndpoint, err := services.minioC.Endpoint(context.Background(), "")
	require.NoError(t, err)

	minioClient, err := minio.New(minioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioUser, minioPassword, ""),
		Secure: false,
	})
	require.NoError(t, err)

	getHandler := NewGetHandler(usecase.NewGetter(retriever))

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

	e.GET(fmt.Sprintf("/:%s", presentation.Sha256Param), getHandler.HandleGet,
		middleware.AuthMiddleware("get"))

	testCases := []struct {
		name           string
		setup          func(t *testing.T, minioClient *minio.Client, db *database.Database, minioEndpoint string, serverURL string) (*http.Request, int64, string, string)
		paramHash      string
		expectedStatus int
		checkResponse  func(t *testing.T, resp *http.Response, expectedSize int64, expectedType string, expectedRedirectURL string)
	}{
		{
			name: "Successful retrieval of blob",
			setup: func(t *testing.T, minioClient *minio.Client, db *database.Database, minioEndpoint string, serverURL string) (*http.Request, int64, string, string) {
				t.Helper()
				content := []byte("test content for get handler")
				hashBytes := sha256.Sum256(content)
				hexHash := hex.EncodeToString(hashBytes[:])
				blobType := "text/plain"
				minioObjectPath := fmt.Sprintf("http://%s/%s/%s",
					minioEndpoint, minioBucket, hexHash)

				_, err := minioClient.PutObject(context.Background(),
					minioBucket, hexHash, bytes.NewReader(content), int64(len(content)),
					minio.PutObjectOptions{ContentType: blobType})
				require.NoError(t, err)

				blob := &model.Blob{
					ID:           hexHash,
					Bucket:       minioBucket,
					MinIOAddress: minioObjectPath,
					UploadTime:   time.Now(),
					Author:       "f2f357f4955a5b51b329381c828d5789f28d88e6e582d921532c25672c833d7b",
					BlobType:     blobType,
					Size:         int64(len(content)),
				}

				coll := db.Client.Database(db.DBName).Collection(database.BlobCollection)
				_, err = coll.InsertOne(context.Background(), blob)
				require.NoError(t, err)

				req := httptest.NewRequest(http.MethodGet, "/"+hexHash, http.NoBody)
				req.Header.Set(presentation.AuthKey, generateValidAuthHeader(t, 600, "get", hexHash, serverURL))

				return req, int64(len(content)), blobType, minioObjectPath
			},
			paramHash:      "",
			expectedStatus: http.StatusFound,
			checkResponse: func(t *testing.T, resp *http.Response, expectedSize int64, expectedType string, expectedRedirectURL string) {
				t.Helper()
				assert.Equal(t, fmt.Sprintf("%d", expectedSize), resp.Header.Get("Content-Length"))
				assert.Equal(t, expectedType, resp.Header.Get("Content-Type"))
				assert.Equal(t, "bytes", resp.Header.Get("Accept-Ranges"))
				assert.Equal(t, expectedRedirectURL, resp.Header.Get("Location"))
			},
		},
		{
			name: "Blob not found for get",
			setup: func(t *testing.T, _ *minio.Client, _ *database.Database, _ string, serverURL string) (*http.Request, int64, string, string) {
				t.Helper()
				nonExistentHash := "nonexistenthashforget"
				req := httptest.NewRequest(http.MethodGet, "/"+nonExistentHash, http.NoBody)
				req.Header.Set(presentation.AuthKey, generateValidAuthHeader(t, 600, "get", nonExistentHash, serverURL))

				return req, 0, "", ""
			},
			paramHash:      "nonexistenthashforget",
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, resp *http.Response, _ int64, _ string, _ string) {
				t.Helper()
				assert.Equal(t, "blob not found", resp.Header.Get("X-Reason"))
			},
		},
		{
			name: "Hash with extension for get",
			setup: func(t *testing.T, minioClient *minio.Client, db *database.Database, minioEndpoint string, serverURL string) (*http.Request, int64, string, string) {
				t.Helper()
				content := []byte("content with extension for get")
				hashBytes := sha256.Sum256(content)
				hexHash := hex.EncodeToString(hashBytes[:])
				blobType := "image/jpeg"
				minioObjectPath := fmt.Sprintf("http://%s/%s/%s", minioEndpoint, minioBucket, hexHash)

				_, err := minioClient.PutObject(context.Background(), minioBucket, hexHash, bytes.NewReader(content), int64(len(content)), minio.PutObjectOptions{ContentType: blobType})
				require.NoError(t, err)

				blob := &model.Blob{
					ID:           hexHash,
					Bucket:       minioBucket,
					MinIOAddress: minioObjectPath,
					UploadTime:   time.Now(),
					Author:       "f2f357f4955a5b51b329381c828d5789f28d88e6e582d921532c25672c833d7b",
					BlobType:     blobType,
					Size:         int64(len(content)),
				}

				coll := db.Client.Database(db.DBName).Collection(database.BlobCollection)
				_, err = coll.InsertOne(context.Background(), blob)
				require.NoError(t, err)

				req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/%s.jpg", hexHash), http.NoBody)
				req.Header.Set(presentation.AuthKey, generateValidAuthHeader(t, 600, "get", hexHash, serverURL))

				return req, int64(len(content)), blobType, minioObjectPath
			},
			paramHash:      ".jpg",
			expectedStatus: http.StatusFound,
			checkResponse: func(t *testing.T, resp *http.Response, expectedSize int64, expectedType string, expectedRedirectURL string) {
				t.Helper()
				assert.Equal(t, fmt.Sprintf("%d", expectedSize), resp.Header.Get("Content-Length"))
				assert.Equal(t, expectedType, resp.Header.Get("Content-Type"))
				assert.Equal(t, expectedRedirectURL, resp.Header.Get("Location"))
			},
		},
		{
			name: "Expired event in Authorization header",
			setup: func(t *testing.T, _ *minio.Client, _ *database.Database, _ string, serverURL string) (*http.Request, int64, string, string) {
				t.Helper()
				hash := "0000000000000000000000000000000000000000000000000000000000000000"
				req := httptest.NewRequest(http.MethodGet, "/"+hash, http.NoBody)
				req.Header.Set(presentation.AuthKey, generateValidAuthHeader(t, -600, "get", hash, serverURL))

				return req, 0, "", ""
			},
			paramHash:      "somehash",
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, resp *http.Response, _ int64, _ string, _ string) {
				t.Helper()
				assert.Equal(t, "invalid expiration", resp.Header.Get("X-Reason"))
			},
		},
		{
			name: "Wrong action in Authorization header",
			setup: func(t *testing.T, _ *minio.Client, _ *database.Database, _ string, serverURL string) (*http.Request, int64, string, string) {
				t.Helper()
				hash := "0000000000000000000000000000000000000000000000000000000000000000"
				req := httptest.NewRequest(http.MethodGet, "/"+hash, http.NoBody)
				req.Header.Set(presentation.AuthKey, generateValidAuthHeader(t, 600, "upload", hash, serverURL))

				return req, 0, "", ""
			},
			paramHash:      "somehash",
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, resp *http.Response, _ int64, _ string, _ string) {
				t.Helper()
				assert.Equal(t, "invalid action", resp.Header.Get("X-Reason"))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			serverURL := "http://localhost:8080"
			req, size, bType, minioAddress := tc.setup(t, minioClient, db, minioEndpoint, serverURL)

			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, tc.expectedStatus, rec.Code)
			tc.checkResponse(t, rec.Result(), size, bType, minioAddress)
		})
	}
}

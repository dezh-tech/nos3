package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"nos3/internal/application/usecase"
	"nos3/internal/domain/dto"
	"nos3/internal/domain/model"
	"nos3/internal/infrastructure/database"
	"nos3/internal/infrastructure/grpcclient"
	"nos3/internal/presentation"
	"nos3/internal/presentation/middleware"

	"github.com/labstack/echo/v4"
	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleList_Integration(t *testing.T) {
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

	dbLister := database.NewBlobLister(db, grpcClient)
	listHandler := NewListHandler(usecase.NewLister(dbLister, "http://localhost:8080"))

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

	e.GET(fmt.Sprintf("/list/:%s", presentation.PK), listHandler.HandleList,
		middleware.AuthMiddleware("list"))

	testAuthor1 := "a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1"

	insertBlob := func(t *testing.T, db *database.Database, id, author string, uploadTime time.Time) *model.Blob {
		t.Helper()
		blob := &model.Blob{
			ID:           id,
			Bucket:       minioBucket,
			MinIOAddress: "minio://test/test",
			UploadTime:   uploadTime,
			Author:       author,
			BlobType:     "text/plain",
			Size:         100,
		}
		coll := db.Client.Database(db.DBName).Collection(database.BlobCollection)
		_, err := coll.InsertOne(context.Background(), blob)
		require.NoError(t, err)

		return blob
	}

	now := time.Now().Truncate(time.Second)

	blob1 := insertBlob(t, db, "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
		testAuthor1, now.Add(-2*time.Hour))
	blob2 := insertBlob(t, db, "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
		testAuthor1, now.Add(-1*time.Hour))
	blob3 := insertBlob(t, db, "0987654321abcdef0987654321abcdef0987654321abcdef0987654321abcdef",
		testAuthor1, now)

	testCases := []struct {
		name            string
		pubkeyParam     string
		sinceParam      string
		untilParam      string
		expectedStatus  int
		expectedBlobs   []dto.BlobDescriptor
		expectedReason  string
		setupAuthHeader func(t *testing.T, action, hash, serverURL string) string
	}{
		{
			name:           "Successfully list blobs for an author",
			pubkeyParam:    testAuthor1,
			sinceParam:     "",
			untilParam:     "",
			expectedStatus: http.StatusOK,
			expectedBlobs: []dto.BlobDescriptor{
				{URL: fmt.Sprintf("http://localhost:8080/%s", blob1.ID), Sha256: blob1.ID, Size: blob1.Size, FileType: blob1.BlobType, Uploaded: blob1.UploadTime.Unix()},
				{URL: fmt.Sprintf("http://localhost:8080/%s", blob2.ID), Sha256: blob2.ID, Size: blob2.Size, FileType: blob2.BlobType, Uploaded: blob2.UploadTime.Unix()},
				{URL: fmt.Sprintf("http://localhost:8080/%s", blob3.ID), Sha256: blob3.ID, Size: blob3.Size, FileType: blob3.BlobType, Uploaded: blob3.UploadTime.Unix()},
			},
		},
		{
			name:           "List blobs for an author with 'since' filter",
			pubkeyParam:    testAuthor1,
			sinceParam:     strconv.FormatInt(now.Add(-1*time.Hour).Unix(), 10),
			untilParam:     "",
			expectedStatus: http.StatusOK,
			expectedBlobs: []dto.BlobDescriptor{
				{URL: fmt.Sprintf("http://localhost:8080/%s", blob2.ID), Sha256: blob2.ID, Size: blob2.Size, FileType: blob2.BlobType, Uploaded: blob2.UploadTime.Unix()},
				{URL: fmt.Sprintf("http://localhost:8080/%s", blob3.ID), Sha256: blob3.ID, Size: blob3.Size, FileType: blob3.BlobType, Uploaded: blob3.UploadTime.Unix()},
			},
		},
		{
			name:           "List blobs for an author with 'until' filter",
			pubkeyParam:    testAuthor1,
			sinceParam:     "",
			untilParam:     strconv.FormatInt(now.Add(-1*time.Hour).Unix(), 10),
			expectedStatus: http.StatusOK,
			expectedBlobs: []dto.BlobDescriptor{
				{URL: fmt.Sprintf("http://localhost:8080/%s", blob1.ID), Sha256: blob1.ID, Size: blob1.Size, FileType: blob1.BlobType, Uploaded: blob1.UploadTime.Unix()},
				{URL: fmt.Sprintf("http://localhost:8080/%s", blob2.ID), Sha256: blob2.ID, Size: blob2.Size, FileType: blob2.BlobType, Uploaded: blob2.UploadTime.Unix()},
			},
		},
		{
			name:           "List blobs for an author with both 'since' and 'until' filters",
			pubkeyParam:    testAuthor1,
			sinceParam:     strconv.FormatInt(now.Add(-1*time.Hour).Unix(), 10),
			untilParam:     strconv.FormatInt(now.Add(-30*time.Minute).Unix(), 10),
			expectedStatus: http.StatusOK,
			expectedBlobs: []dto.BlobDescriptor{
				{URL: fmt.Sprintf("http://localhost:8080/%s", blob2.ID), Sha256: blob2.ID, Size: blob2.Size, FileType: blob2.BlobType, Uploaded: blob2.UploadTime.Unix()},
			},
		},
		{
			name:           "List blobs for an author with no matching blobs",
			pubkeyParam:    "nonexistentpubkey00000000000000000000000000000000000000000000000000000000",
			sinceParam:     "",
			untilParam:     "",
			expectedStatus: http.StatusOK,
			expectedBlobs:  []dto.BlobDescriptor{},
		},
		{
			name:           "Invalid 'since' query parameter",
			pubkeyParam:    testAuthor1,
			sinceParam:     "invalid-timestamp",
			untilParam:     "",
			expectedStatus: http.StatusBadRequest,
			expectedReason: "invalid 'since' timestamp",
		},
		{
			name:           "Invalid 'until' query parameter",
			pubkeyParam:    testAuthor1,
			sinceParam:     "",
			untilParam:     "bad-timestamp",
			expectedStatus: http.StatusBadRequest,
			expectedReason: "invalid 'until' timestamp",
		},
		{
			name:           "Missing pubkey in URL (handled by Echo)",
			pubkeyParam:    "",
			sinceParam:     "",
			untilParam:     "",
			expectedStatus: http.StatusNotFound,
			expectedReason: "",
		},
		{
			name:           "Unauthorized access: missing Auth header",
			pubkeyParam:    testAuthor1,
			sinceParam:     "",
			untilParam:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedReason: "missing Authorization header",
			setupAuthHeader: func(t *testing.T, _, _, _ string) string {
				t.Helper()

				return ""
			},
		},
		{
			name:           "Unauthorized access: wrong action in Auth header",
			pubkeyParam:    testAuthor1,
			sinceParam:     "",
			untilParam:     "",
			expectedStatus: http.StatusUnauthorized,
			expectedReason: "invalid action",
			setupAuthHeader: func(t *testing.T, _, hash, serverURL string) string {
				t.Helper()

				return generateValidAuthHeader(t, 600, "upload", hash, serverURL)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetURL := fmt.Sprintf("/list/%s", tc.pubkeyParam)
			req := httptest.NewRequest(http.MethodGet, targetURL, http.NoBody)

			var authHeader string
			if tc.setupAuthHeader != nil {
				authHeader = tc.setupAuthHeader(t, "list", "", "http://localhost:8080")
			} else {
				authHeader = generateValidAuthHeader(t, 600, "list", "", "http://localhost:8080")
			}
			if authHeader != "" {
				req.Header.Set(presentation.AuthKey, authHeader)
			}

			q := req.URL.Query()
			if tc.sinceParam != "" {
				q.Set("since", tc.sinceParam)
			}
			if tc.untilParam != "" {
				q.Set("until", tc.untilParam)
			}
			req.URL.RawQuery = q.Encode()

			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, tc.expectedStatus, rec.Code)
			if tc.expectedStatus == http.StatusOK {
				var receivedBlobs []dto.BlobDescriptor
				err := json.NewDecoder(rec.Body).Decode(&receivedBlobs)
				require.NoError(t, err)
				assert.ElementsMatch(t, tc.expectedBlobs, receivedBlobs)
			} else if tc.expectedReason != "" {
				assert.Contains(t, rec.Header().Get(presentation.ReasonTag), tc.expectedReason)
			}
		})
	}
}

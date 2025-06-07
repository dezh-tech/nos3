package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"nos3/pkg/logger"
	"strconv"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/nbd-wtf/go-nostr"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"

	"nos3/internal/domain/dto"
	"nos3/internal/presentation"

	echoMiddleware "github.com/labstack/echo/v4/middleware"

	"nos3/internal/application/usecase"
	"nos3/internal/infrastructure/broker"
	"nos3/internal/infrastructure/database"
	"nos3/internal/infrastructure/grpcclient"
	"nos3/internal/infrastructure/grpcclient/gen"
	"nos3/internal/presentation/middleware"

	minioInfra "nos3/internal/infrastructure/minio"
)

const (
	minioImage    = "minio/minio:latest"
	minioUser     = "minioadmin"
	minioPassword = "minioadmin"
	minioBucket   = "test-bucket"

	mongoImage    = "mongo:latest"
	mongoUser     = "testuser"
	mongoPassword = "testpass"
	mongoDBName   = "testdb"

	redisImage = "redis:7-alpine"
	SecretKey  = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

type testServices struct {
	minioClient *minio.Client
	mongoClient *mongo.Client
	redisClient *redis.Client
	minioC      testcontainers.Container
	mongoC      testcontainers.Container
	redisC      testcontainers.Container
}

func setupServices(t *testing.T) *testServices {
	t.Helper()

	ctx := context.Background()

	minioReq := testcontainers.ContainerRequest{
		Image:        minioImage,
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     minioUser,
			"MINIO_ROOT_PASSWORD": minioPassword,
		},
		Cmd:        []string{"server", "/data"},
		WaitingFor: wait.ForHTTP("/minio/health/live").WithPort("9000"),
	}
	minioC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: minioReq,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to start MinIO container: %v", err)
	}

	minioEndpoint, err := minioC.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("Failed to get MinIO endpoint: %v", err)
	}

	minioClient, err := minio.New(minioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioUser, minioPassword, ""),
		Secure: false,
	})
	if err != nil {
		t.Fatalf("Failed to create MinIO client: %v", err)
	}

	err = minioClient.MakeBucket(ctx, minioBucket, minio.MakeBucketOptions{})
	if err != nil {
		t.Fatalf("Failed to create MinIO bucket: %v", err)
	}

	mongoReq := testcontainers.ContainerRequest{
		Image:        mongoImage,
		ExposedPorts: []string{"27017/tcp"},
		Env: map[string]string{
			"MONGO_INITDB_ROOT_USERNAME": mongoUser,
			"MONGO_INITDB_ROOT_PASSWORD": mongoPassword,
		},
		WaitingFor: wait.ForLog("Waiting for connections").WithStartupTimeout(30 * time.Second),
	}
	mongoC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: mongoReq,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to start MongoDB container: %v", err)
	}

	mongoEndpoint, err := mongoC.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("Failed to get MongoDB endpoint: %v", err)
	}

	mongoURI := fmt.Sprintf("mongodb://%s:%s@%s", mongoUser, mongoPassword, mongoEndpoint)
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	redisReq := testcontainers.ContainerRequest{
		Image:        redisImage,
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp"),
	}
	redisC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: redisReq,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("Failed to start Redis container: %v", err)
	}

	redisEndpoint, err := redisC.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("Failed to get Redis endpoint: %v", err)
	}

	redisOpts, err := redis.ParseURL(fmt.Sprintf("redis://%s", redisEndpoint))
	if err != nil {
		t.Fatalf("Failed to parse Redis URL: %v", err)
	}

	redisClient := redis.NewClient(redisOpts)

	return &testServices{
		minioClient: minioClient,
		mongoClient: mongoClient,
		redisClient: redisClient,
		minioC:      minioC,
		mongoC:      mongoC,
		redisC:      redisC,
	}
}

func cleanupServices(t *testing.T, s *testServices) {
	t.Helper()
	ctx := context.Background()

	if err := s.minioC.Terminate(ctx); err != nil {
		t.Errorf("Failed to terminate MinIO container: %v", err)
	}
	if err := s.mongoC.Terminate(ctx); err != nil {
		t.Errorf("Failed to terminate MongoDB container: %v", err)
	}
	if err := s.redisC.Terminate(ctx); err != nil {
		t.Errorf("Failed to terminate Redis container: %v", err)
	}
}

func startTestGRPCServer(t *testing.T) (string, func()) {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	gen.RegisterLogServer(server, &mockService{})
	go func() {
		require.NoError(t, server.Serve(lis))
	}()
	cleanup := func() {
		server.GracefulStop()
	}

	return lis.Addr().String(), cleanup
}

type mockService struct {
	gen.UnimplementedLogServer
}

func (m *mockService) AddLog(context.Context, *gen.AddLogRequest) (*gen.AddLogResponse, error) {
	return &gen.AddLogResponse{
		Success: true,
	}, nil
}

func (m *mockService) RegisterService(context.Context, *gen.RegisterServiceRequest, ...grpc.CallOption) (*gen.RegisterServiceResponse, error) {
	return &gen.RegisterServiceResponse{
		Success: true,
	}, nil
}

func TestHandle_Integration(t *testing.T) {
	services := setupServices(t)
	defer cleanupServices(t, services)

	addr, cleanup := startTestGRPCServer(t)
	defer cleanup()
	grpcClient, err := grpcclient.New(grpcclient.ClientConfig{
		Endpoint:  addr,
		Heartbeat: 30,
	})
	if err != nil {
		t.Fatal(err)
	}

	redisClient, err := broker.NewClient(broker.Config{
		URI:        "redis://" + services.redisClient.Options().Addr,
		StreamName: "test-stream",
		GroupName:  "test-group",
	}, grpcClient)
	if err != nil {
		t.Fatal(err)
	}

	publisher := broker.NewPublisher(redisClient, broker.PublisherConfig{Timeout: 1000}, grpcClient)

	mongoEndpoint, err := services.mongoC.Endpoint(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}

	db, err := database.Connect(database.Config{
		URI:               fmt.Sprintf("mongodb://%s:%s@%s", mongoUser, mongoPassword, mongoEndpoint),
		DBName:            mongoDBName,
		ConnectionTimeout: 30000,
		QueryTimeout:      30000,
	}, grpcClient)
	if err != nil {
		t.Fatal(err)
	}
	defer func(db *database.Database) {
		err := db.Stop()
		if err != nil {
			logger.Error("couldn't stop db instance")
		}
	}(db)

	writer := database.NewBlobWriter(db, grpcClient)
	retriever := database.NewBlobRetriever(db, grpcClient)
	endpoint, err := services.minioC.Endpoint(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioUser, minioPassword, ""),
		Secure: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	handler := NewUploadHandler(usecase.NewUploader(
		publisher,
		retriever,
		writer,
		minioInfra.NewUploader(minioClient, grpcClient, minioInfra.UploaderConfig{
			Timeout: 3000,
			Bucket:  minioBucket,
		}),
		minioInfra.NewRemover(minioClient, grpcClient, minioInfra.RemoverConfig{Timeout: 3000}),
		database.NewRemover(db, grpcClient),
		"http://localhost:8080",
	))

	e := echo.New()
	e.Use(echoMiddleware.Logger())
	e.POST("/upload", handler.Handle, middleware.AuthMiddleware("upload"))
	testCases := []struct {
		name           string
		setupRequest   func() *http.Request
		expectedStatus int
		checkResponse  func(t *testing.T, resp *http.Response)
	}{
		{
			name: "Valid upload request",
			setupRequest: func() *http.Request {
				content := []byte("test content")

				hash := sha256.Sum256(content)
				hexHash := hex.EncodeToString(hash[:])

				bodyReader := io.NopCloser(bytes.NewReader(content))
				req := httptest.NewRequest(http.MethodPost, "/upload", bodyReader)

				req.Header.Set(presentation.AuthKey, generateValidAuthHeader(t,
					600, "upload", hexHash, ""))
				req.Header.Set(presentation.TypeKey, "text/plain")

				return req
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				t.Helper()
				var result dto.BlobDescriptor
				err := json.NewDecoder(resp.Body).Decode(&result)
				require.NoError(t, err)
				assert.NotEmpty(t, result.URL)
				assert.NotEmpty(t, result.Sha256)
			},
		},
		{
			name: "Large file upload (10MB)",
			setupRequest: func() *http.Request {
				content := bytes.Repeat([]byte("a"), 10*1024*1024)
				hash := sha256.Sum256(content)
				req := httptest.NewRequest(http.MethodPost, "/upload", io.NopCloser(bytes.NewReader(content)))
				req.Header.Set(presentation.AuthKey, generateValidAuthHeader(t, 600, "upload", hex.EncodeToString(hash[:]), ""))
				req.Header.Set(presentation.TypeKey, "text/plain")

				return req
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				t.Helper()
				var result dto.BlobDescriptor
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
				assert.Equal(t, int64(10*1024*1024), result.Size)
			},
		},
		{
			name: "PDF file upload",
			setupRequest: func() *http.Request {
				content := append([]byte("%PDF-"), bytes.Repeat([]byte("a"), 1024)...)
				hash := sha256.Sum256(content)
				req := httptest.NewRequest(http.MethodPost, "/upload", io.NopCloser(bytes.NewReader(content)))
				req.Header.Set(presentation.AuthKey, generateValidAuthHeader(t, 600, "upload", hex.EncodeToString(hash[:]), ""))
				req.Header.Set(presentation.TypeKey, "application/pdf")

				return req
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				t.Helper()
				var result dto.BlobDescriptor
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
				assert.Equal(t, "application/pdf", result.FileType)
			},
		},
		{
			name: "Image upload (PNG)",
			setupRequest: func() *http.Request {
				content := append([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, bytes.Repeat([]byte("a"), 100)...)
				hash := sha256.Sum256(content)
				req := httptest.NewRequest(http.MethodPost, "/upload", io.NopCloser(bytes.NewReader(content)))
				req.Header.Set(presentation.AuthKey, generateValidAuthHeader(t, 600, "upload", hex.EncodeToString(hash[:]), ""))
				req.Header.Set(presentation.TypeKey, "image/png")

				return req
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				t.Helper()
				var result dto.BlobDescriptor
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
				assert.Equal(t, "image/png", result.FileType)
			},
		},
		{
			name: "Missing Authorization header",
			setupRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader([]byte("test")))
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Invalid Nostr prefix",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader([]byte("test")))
				req.Header.Set(presentation.AuthKey, "Bearer invalid")

				return req
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Expired event",
			setupRequest: func() *http.Request {
				content := []byte("test")
				hash := sha256.Sum256(content)
				req := httptest.NewRequest(http.MethodPost, "/upload", io.NopCloser(bytes.NewReader(content)))
				req.Header.Set(presentation.AuthKey, generateValidAuthHeader(t, -600, "upload", hex.EncodeToString(hash[:]), ""))

				return req
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Wrong action",
			setupRequest: func() *http.Request {
				content := []byte("test")
				hash := sha256.Sum256(content)
				req := httptest.NewRequest(http.MethodPost, "/upload", io.NopCloser(bytes.NewReader(content)))
				req.Header.Set(presentation.AuthKey, generateValidAuthHeader(t, 600, "download", hex.EncodeToString(hash[:]), ""))

				return req
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Invalid hash",
			setupRequest: func() *http.Request {
				content := []byte("test")
				req := httptest.NewRequest(http.MethodPost, "/upload", io.NopCloser(bytes.NewReader(content)))
				req.Header.Set(presentation.AuthKey, generateValidAuthHeader(t, 600, "upload", "invalidhash", ""))

				return req
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Empty body",
			setupRequest: func() *http.Request {
				content := []byte("")
				hash := sha256.Sum256(content)
				req := httptest.NewRequest(http.MethodPost, "/upload", io.NopCloser(bytes.NewReader(content)))
				req.Header.Set(presentation.AuthKey, generateValidAuthHeader(t, 600, "upload", hex.EncodeToString(hash[:]), ""))

				return req
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Wrong HTTP method (GET)",
			setupRequest: func() *http.Request {
				content := []byte("test")
				hash := sha256.Sum256(content)
				req := httptest.NewRequest(http.MethodGet, "/upload", io.NopCloser(bytes.NewReader(content)))
				req.Header.Set(presentation.AuthKey, generateValidAuthHeader(t, 600, "upload", hex.EncodeToString(hash[:]), ""))

				return req
			},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name: "Invalid Content-Type",
			setupRequest: func() *http.Request {
				content := []byte("test")
				hash := sha256.Sum256(content)
				req := httptest.NewRequest(http.MethodPost, "/upload", io.NopCloser(bytes.NewReader(content)))
				req.Header.Set(presentation.AuthKey, generateValidAuthHeader(t, 600, "upload", hex.EncodeToString(hash[:]), ""))
				req.Header.Set(presentation.TypeKey, "invalid/type")

				return req
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Video upload (MP4)",
			setupRequest: func() *http.Request {
				content := append([]byte("\x00\x00\x00\x18ftypmp42"), bytes.Repeat([]byte("a"), 5*1024*1024)...)
				hash := sha256.Sum256(content)
				req := httptest.NewRequest(http.MethodPost, "/upload", io.NopCloser(bytes.NewReader(content)))
				req.Header.Set(presentation.AuthKey, generateValidAuthHeader(t, 600, "upload", hex.EncodeToString(hash[:]), ""))
				req.Header.Set(presentation.TypeKey, "video/mp4")

				return req
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				t.Helper()
				var result dto.BlobDescriptor
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
				assert.Equal(t, "video/mp4", result.FileType)
			},
		},
		{
			name: "Compressed file (ZIP)",
			setupRequest: func() *http.Request {
				content := append([]byte("PK\x03\x04"), bytes.Repeat([]byte("a"), 1024)...)
				hash := sha256.Sum256(content)
				req := httptest.NewRequest(http.MethodPost, "/upload", io.NopCloser(bytes.NewReader(content)))
				req.Header.Set(presentation.AuthKey, generateValidAuthHeader(t, 600, "upload", hex.EncodeToString(hash[:]), ""))
				req.Header.Set(presentation.TypeKey, "application/zip")

				return req
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				t.Helper()
				var result dto.BlobDescriptor
				require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
				assert.Equal(t, "application/zip", result.FileType)
			},
		},
		{
			name: "Invalid JSON event",
			setupRequest: func() *http.Request {
				badJSON := base64.StdEncoding.EncodeToString([]byte(`{"invalid":`))
				req := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewReader([]byte("test")))
				req.Header.Set(presentation.AuthKey, "Nostr "+badJSON)

				return req
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "Replay attack protection",
			setupRequest: func() *http.Request {
				content := []byte("test")
				hash := sha256.Sum256(content)
				authHeader := generateValidAuthHeader(t, 600, "upload", hex.EncodeToString(hash[:]), "")

				req1 := httptest.NewRequest(http.MethodPost, "/upload", io.NopCloser(bytes.NewReader(content)))
				req1.Header.Set(presentation.AuthKey, authHeader)
				req1.Header.Set(presentation.TypeKey, "text/plain")
				rec1 := httptest.NewRecorder()
				e.ServeHTTP(rec1, req1)
				if rec1.Code != http.StatusOK {
					t.Fatal("First request failed")
				}

				req2 := httptest.NewRequest(http.MethodPost, "/upload", io.NopCloser(bytes.NewReader(content)))
				req2.Header.Set(presentation.AuthKey, authHeader)
				req2.Header.Set(presentation.TypeKey, "text/plain")

				return req2
			},
			expectedStatus: http.StatusBadRequest,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.setupRequest()
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, tc.expectedStatus, rec.Code)
			if tc.checkResponse != nil {
				tc.checkResponse(t, rec.Result())
			}
		})
	}
}

func generateValidAuthHeader(t *testing.T, expirationOffset int64, action, hash, serverURL string) string {
	t.Helper()

	tags := nostr.Tags{
		{presentation.ExpTag, strconv.FormatInt(time.Now().Unix()+expirationOffset, 10)},
		{presentation.TTag, action},
		{presentation.XTag, hash},
	}

	if serverURL != "" && action == "get" {
		tags = append(tags, nostr.Tag{presentation.ServerTag, serverURL})
	}

	event := nostr.Event{
		Kind:      24242,
		CreatedAt: nostr.Timestamp(time.Now().Unix() - 10),
		Tags:      tags,
	}
	_ = event.Sign(SecretKey)
	eventBytes, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}

	return "Nostr " + base64.StdEncoding.EncodeToString(eventBytes)
}

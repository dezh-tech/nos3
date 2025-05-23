package handler

import (
	"context"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"

	echoMiddleware "github.com/labstack/echo/v4/middleware"
	"net"
	"nos3/internal/application/usecase"
	"nos3/internal/infrastructure/broker"
	"nos3/internal/infrastructure/database"
	"nos3/internal/infrastructure/grpcclient"
	"nos3/internal/infrastructure/grpcclient/gen"
	"nos3/internal/presentation/middleware"

	minioInfra "nos3/internal/infrastructure/minio"
	"testing"
	"time"
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

func startTestGRPCServer(t *testing.T) (addr string, cleanup func()) {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	gen.RegisterLogServer(server, &mockService{})
	go func() {
		require.NoError(t, server.Serve(lis))
	}()
	cleanup = func() {
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
	grpcClient, err := grpcclient.New(addr, grpcclient.Config{
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

	handler := &UploadHandler{
		uploader: usecase.NewUploader(
			publisher,
			retriever,
			writer,
			minioInfra.NewUploader(minioClient, grpcClient, &minioInfra.UploaderConfig{
				Timeout: 3000,
				Bucket:  minioBucket,
			}),
			minioInfra.NewRemover(minioClient, grpcClient, &minioInfra.RemoverConfig{Timeout: 3000}),
			database.NewRemover(db, grpcClient),
			"http://localhost:8080",
		),
	}

	e := echo.New()
	e.Use(echoMiddleware.Logger())
	e.POST("/", handler.Handle, middleware.AuthMiddleware("upload"))

}

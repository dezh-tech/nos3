package database

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"nos3/internal/domain/model"
)

const (
	TestUsername = "testuser"
	TestPassword = "testpass"
	TestDBName   = "testdb"
)

func setupMongo(t *testing.T) string {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "mongo:latest",
		ExposedPorts: []string{"27017/tcp"},
		Env: map[string]string{
			"MONGO_INITDB_ROOT_USERNAME": TestUsername,
			"MONGO_INITDB_ROOT_PASSWORD": TestPassword,
		},
		WaitingFor: wait.ForLog("Waiting for connections").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatal("Failed to start MongoDB container:", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal("Failed to get container host:", err)
	}

	port, err := container.MappedPort(ctx, "27017")
	if err != nil {
		t.Fatal("Failed to get mapped port:", err)
	}

	hostPort := net.JoinHostPort(host, port.Port())
	uri := fmt.Sprintf("mongodb://%s:%s@%s", TestUsername, TestPassword, hostPort)

	clientOpts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		t.Fatal("Failed to create MongoDB client:", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		t.Fatal("Failed to ping MongoDB:", err)
	}

	return uri
}

func TestWrite(t *testing.T) {
	t.Parallel()
	uri := setupMongo(t)

	db, err := Connect(Config{
		URI:               uri,
		DBName:            TestDBName,
		ConnectionTimeout: 30000,
		QueryTimeout:      30000,
	})
	require.NoError(t, err)

	writer := NewMediaWriter(db)

	baseMedia := &model.Media{
		ID:           "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		MinIOAddress: "minio://bucket/key",
		UploadTime:   time.Now(),
		Author:       "npub11111111111111111111111111111111111111111111111111111111111",
		MediaType:    "image/png",
	}

	tests := []struct {
		name        string
		modify      func(m *model.Media)
		expectError string
	}{
		{
			name:        "valid media",
			modify:      func(_ *model.Media) {},
			expectError: "",
		},
		{
			name: "missing required author",
			modify: func(m *model.Media) {
				m.Author = ""
			},
			expectError: "Document failed validation",
		},
		{
			name: "invalid _id length",
			modify: func(m *model.Media) {
				m.ID = "short"
			},
			expectError: "Document failed validation",
		},
		{
			name: "invalid author pattern",
			modify: func(m *model.Media) {
				m.Author = "user123"
			},
			expectError: "Document failed validation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			copyMedia := *baseMedia
			tt.modify(&copyMedia)

			err := writer.Write(context.Background(), &copyMedia)

			if tt.expectError == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.expectError)
			}
		})
	}
}

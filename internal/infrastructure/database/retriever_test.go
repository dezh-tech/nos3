package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"nos3/internal/domain/model"
	"nos3/internal/infrastructure/grpcclient"
	"nos3/internal/infrastructure/grpcclient/gen"
)

type MockGRPC struct {
	mock.Mock
}

func (m *MockGRPC) RegisterService(_ context.Context, _, _ string) (*gen.RegisterServiceResponse, error) {
	return &gen.RegisterServiceResponse{}, nil
}

func (m *MockGRPC) AddLog(_ context.Context, _, _ string) (*gen.AddLogResponse, error) {
	return &gen.AddLogResponse{}, nil
}

func TestRetrieve(t *testing.T) {
	t.Parallel()
	uri := setupMongo(t)

	db, err := Connect(Config{
		URI:               uri,
		DBName:            TestDBName,
		ConnectionTimeout: 30000,
		QueryTimeout:      30000,
	}, &grpcclient.Client{})
	require.NoError(t, err)

	retriever := NewBlobRetriever(db, &MockGRPC{})

	ctx := context.Background()
	coll := db.Client.Database(TestDBName).Collection(BlobCollection)

	expectedID := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	expectedBlob := &model.Blob{
		ID:           "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		MinIOAddress: "minio://bucket/key",
		UploadTime:   time.Now(),
		Author:       "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		BlobType:     "image/png",
	}

	_, err = coll.InsertOne(ctx, expectedBlob)
	require.NoError(t, err)

	got, err := retriever.GetByID(ctx, expectedID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, expectedBlob.ID, got.ID)
	require.Equal(t, expectedBlob.MinIOAddress, got.MinIOAddress)
}

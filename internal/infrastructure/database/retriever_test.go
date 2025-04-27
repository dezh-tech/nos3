package database

import (
	"context"
	"testing"
	"time"

	"nos3/internal/domain/model"

	"github.com/stretchr/testify/require"
)

func TestRetrieve(t *testing.T) {
	t.Parallel()
	uri := setupMongo(t)

	db, err := Connect(Config{
		URI:               uri,
		DBName:            TestDBName,
		ConnectionTimeout: 30000,
		QueryTimeout:      30000,
	})
	require.NoError(t, err)

	retriever := NewBlobRetriever(db)

	ctx := context.Background()
	coll := db.Client.Database(TestDBName).Collection(BlobCollection)

	expectedID := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	expectedBlob := &model.Blob{
		ID:           "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		MinIOAddress: "minio://bucket/key",
		UploadTime:   time.Now(),
		Author:       "npub11111111111111111111111111111111111111111111111111111111111",
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

package database

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"nos3/internal/domain/model"
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

	retriever := NewMediaRetriever(db)

	ctx := context.Background()
	coll := db.Client.Database(TestDBName).Collection(MediaCollection)

	expectedID := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	expectedMedia := &model.Media{
		ID:           "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		MinIOAddress: "minio://bucket/key",
		UploadTime:   time.Now(),
		Author:       "npub11111111111111111111111111111111111111111111111111111111111",
		MediaType:    "image/png",
	}

	_, err = coll.InsertOne(ctx, expectedMedia)
	require.NoError(t, err)

	got, err := retriever.GetByID(ctx, expectedID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, expectedMedia.ID, got.ID)
	require.Equal(t, expectedMedia.MinIOAddress, got.MinIOAddress)
}

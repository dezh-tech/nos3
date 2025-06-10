package database

import (
	"context"
	"nos3/pkg/logger"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"

	"nos3/internal/domain/model"
	"nos3/internal/infrastructure/grpcclient/gen"
)

func TestGetByAuthor(t *testing.T) {
	t.Parallel()

	testAuthor1 := "a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1a1"
	testAuthor2 := "b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2b2"

	tests := []struct {
		name          string
		author        string
		getSince      func(time.Time) *time.Time
		getUntil      func(time.Time) *time.Time
		expectedCount int
		expectedIDs   []string
	}{
		{
			name:          "Retrieve all blobs for author1",
			author:        testAuthor1,
			getSince:      func(time.Time) *time.Time { return nil },
			getUntil:      func(time.Time) *time.Time { return nil },
			expectedCount: 3,
			expectedIDs:   []string{"0000000000000000000000000000000000000000000000000000000000000001", "0000000000000000000000000000000000000000000000000000000000000002", "0000000000000000000000000000000000000000000000000000000000000004"},
		},
		{
			name:          "Retrieve blobs for author1 with 'since' filter",
			author:        testAuthor1,
			getSince:      func(now time.Time) *time.Time { return ptrTime(now.Add(-1*time.Hour + 1*time.Second)) },
			getUntil:      func(time.Time) *time.Time { return nil },
			expectedCount: 1,
			expectedIDs:   []string{"0000000000000000000000000000000000000000000000000000000000000004"},
		},
		{
			name:          "Retrieve blobs for author1 with 'until' filter",
			author:        testAuthor1,
			getSince:      func(time.Time) *time.Time { return nil },
			getUntil:      func(now time.Time) *time.Time { return ptrTime(now.Add(-1*time.Hour - 1*time.Second)) },
			expectedCount: 1,
			expectedIDs:   []string{"0000000000000000000000000000000000000000000000000000000000000001"},
		},
		{
			name:          "Retrieve blobs for author1 with both 'since' and 'until' filters",
			author:        testAuthor1,
			getSince:      func(now time.Time) *time.Time { return ptrTime(now.Add(-1*time.Hour + 1*time.Second)) },
			getUntil:      func(now time.Time) *time.Time { return ptrTime(now.Add(-1 * time.Second)) },
			expectedCount: 0,
			expectedIDs:   []string{},
		},
		{
			name:          "Retrieve blobs for author2",
			author:        testAuthor2,
			getSince:      func(time.Time) *time.Time { return nil },
			getUntil:      func(time.Time) *time.Time { return nil },
			expectedCount: 1,
			expectedIDs:   []string{"0000000000000000000000000000000000000000000000000000000000000003"},
		},
		{
			name:          "Retrieve blobs for non-existent author",
			author:        "nonexistentauthor",
			getSince:      func(time.Time) *time.Time { return nil },
			getUntil:      func(time.Time) *time.Time { return nil },
			expectedCount: 0,
			expectedIDs:   []string{},
		},
		{
			name:          "No blobs within time range for author1",
			author:        testAuthor1,
			getSince:      func(now time.Time) *time.Time { return ptrTime(now.Add(1 * time.Hour)) },
			getUntil:      func(now time.Time) *time.Time { return ptrTime(now.Add(2 * time.Hour)) },
			expectedCount: 0,
			expectedIDs:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uri, cleanUp := setupMongo(t)
			defer cleanUp()

			mockGrpcClient := &MockGRPC{}
			mockGrpcClient.On("AddLog", mock.Anything, mock.Anything, mock.Anything).Return(&gen.AddLogResponse{Success: true}, nil).Maybe()

			db, err := Connect(Config{
				URI:               uri,
				DBName:            TestDBName,
				ConnectionTimeout: 30000,
				QueryTimeout:      30000,
			}, mockGrpcClient)
			require.NoError(t, err)
			defer func(db *Database) {
				err := db.Stop()
				if err != nil {
					logger.Error("couldn't stop the db connection")
				}
			}(db)

			lister := NewBlobLister(db, mockGrpcClient)
			ctx := context.Background()
			coll := db.Client.Database(TestDBName).Collection(BlobCollection)

			now := time.Now().Truncate(time.Second)

			blobsToInsert := []*model.Blob{
				{ID: "0000000000000000000000000000000000000000000000000000000000000001", Bucket: "b", MinIOAddress: "m", UploadTime: now.Add(-2 * time.Hour), Author: testAuthor1, BlobType: "t", Size: 1},
				{ID: "0000000000000000000000000000000000000000000000000000000000000002", Bucket: "b", MinIOAddress: "m", UploadTime: now.Add(-1 * time.Hour), Author: testAuthor1, BlobType: "t", Size: 2},
				{ID: "0000000000000000000000000000000000000000000000000000000000000003", Bucket: "b", MinIOAddress: "m", UploadTime: now.Add(-3 * time.Hour), Author: testAuthor2, BlobType: "t", Size: 3},
				{ID: "0000000000000000000000000000000000000000000000000000000000000004", Bucket: "b", MinIOAddress: "m", UploadTime: now, Author: testAuthor1, BlobType: "t", Size: 4},
			}

			for _, blob := range blobsToInsert {
				_, err := coll.InsertOne(ctx, blob)
				require.NoError(t, err)
			}

			defer func() {
				_, err := coll.DeleteMany(context.Background(), bson.M{})
				require.NoError(t, err)
			}()

			since := tt.getSince(now)
			until := tt.getUntil(now)

			gotBlobs, err := lister.GetByAuthor(ctx, tt.author, since, until)
			require.NoError(t, err)
			assert.Len(t, gotBlobs, tt.expectedCount)

			var gotIDs []string
			for _, blob := range gotBlobs {
				gotIDs = append(gotIDs, blob.ID)
			}
			assert.ElementsMatch(t, tt.expectedIDs, gotIDs)
		})
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

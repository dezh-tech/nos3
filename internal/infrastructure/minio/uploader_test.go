package minio

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	TestAccessKey = "minioadmin"
	TestSecretKey = "minioadmin"
	BucketName    = "temp-bucket-for-tests"
)

func setupMinio(t *testing.T) (testcontainers.Container, *minio.Client) {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "minio/minio:latest",
		ExposedPorts: []string{"9000/tcp"},
		Env: map[string]string{
			"MINIO_ROOT_USER":     TestAccessKey,
			"MINIO_ROOT_PASSWORD": TestSecretKey,
		},
		Cmd:        []string{"server", "/data"},
		WaitingFor: wait.ForHTTP("/minio/health/live").WithPort("9000"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatal("Failed to start container:", err)
	}

	endpoint, err := container.Endpoint(ctx, "")
	if err != nil {
		t.Fatal(err)
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:           credentials.NewStaticV4(TestAccessKey, TestSecretKey, ""),
		Secure:          false,
		TrailingHeaders: true,
	})
	if err != nil {
		t.Fatal("Failed to create minio client:", err)
	}

	err = client.MakeBucket(ctx, BucketName, minio.MakeBucketOptions{})
	if err != nil {
		t.Fatal("Failed to create bucket:", err)
	}

	return container, client
}

type uploadIntegrationTestCase struct {
	name             string
	content          []byte
	fileSize         int64
	fileHash         string
	fileType         string
	expectError      bool
	expectedErrorMsg string
	expectedSize     int64
	expectedType     string
	expectFinalName  bool
	simulateCorrupt  bool
}

type corruptReader struct {
	source []byte
	failAt int
	read   int
}

func (r *corruptReader) Read(p []byte) (int, error) {
	if r.read >= r.failAt {
		return 0, errors.New("simulated read error")
	}
	n := copy(p, r.source[r.read:])
	r.read += n

	return n, nil
}

func TestUploadFile(t *testing.T) {
	container, client := setupMinio(t)
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	uploader := NewUploader(client, &Config{
		Timeout: 3000,
		Bucket:  BucketName,
	})

	smallFile := []byte("hello, world!")
	smallHash := sha256.Sum256(smallFile)
	largeFile := bytes.Repeat([]byte("x"), 1024*1024*17) // 17MB
	largeHash := sha256.Sum256(largeFile)

	tests := []uploadIntegrationTestCase{
		{
			name:            "small valid file",
			content:         smallFile,
			fileSize:        int64(len(smallFile)),
			fileHash:        hex.EncodeToString(smallHash[:]),
			fileType:        "text/plain",
			expectError:     false,
			expectedSize:    int64(len(smallFile)),
			expectedType:    "text/plain",
			expectFinalName: true,
		},
		{
			name:            "large file multiple chunks",
			content:         largeFile,
			fileSize:        int64(len(largeFile)),
			fileHash:        hex.EncodeToString(largeHash[:]),
			fileType:        "text/plain",
			expectError:     false,
			expectedSize:    int64(len(largeFile)),
			expectedType:    "text/plain",
			expectFinalName: true,
		},
		{
			name:             "mime type mismatch",
			content:          smallFile,
			fileSize:         int64(len(smallFile)),
			fileHash:         hex.EncodeToString(smallHash[:]),
			fileType:         "image/png",
			expectError:      true,
			expectedErrorMsg: "invalid file type",
		},
		{
			name:             "hash mismatch",
			content:          smallFile,
			fileSize:         int64(len(smallFile)),
			fileHash:         strings.Repeat("0", 64),
			fileType:         "text/plain",
			expectError:      true,
			expectedErrorMsg: "invalid hash",
		},
		{
			name:             "file size mismatch",
			content:          smallFile,
			fileSize:         int64(len(smallFile)) + 5,
			fileHash:         hex.EncodeToString(smallHash[:]),
			fileType:         "text/plain",
			expectError:      true,
			expectedErrorMsg: "file size mismatch",
		},
		{
			name:             "simulate corrupted stream",
			content:          smallFile,
			fileSize:         int64(len(smallFile)),
			fileHash:         hex.EncodeToString(smallHash[:]),
			fileType:         "text/plain",
			simulateCorrupt:  true,
			expectError:      true,
			expectedErrorMsg: "simulated read error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var reader io.ReadCloser
			if tc.simulateCorrupt {
				reader = io.NopCloser(&corruptReader{
					source: tc.content,
					failAt: 5,
				})
			} else {
				reader = io.NopCloser(bytes.NewReader(tc.content))
			}

			result, err := uploader.UploadFile(context.Background(), reader, tc.fileSize, tc.fileHash, tc.fileType)

			if tc.expectError {
				assert.Error(t, err)
				if tc.expectedErrorMsg != "" {
					assert.Contains(t, err.Error(), tc.expectedErrorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedSize, result.Size)
				assert.Contains(t, result.Type, tc.expectedType)
				if tc.expectFinalName {
					_, err := client.StatObject(context.Background(), BucketName, tc.fileHash, minio.StatObjectOptions{})
					assert.NoError(t, err, "expected object %s to exist in MinIO", tc.fileHash)
				}
			}
		})
	}
}

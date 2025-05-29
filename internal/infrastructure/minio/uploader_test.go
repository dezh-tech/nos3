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

	"github.com/stretchr/testify/mock"

	"nos3/internal/infrastructure/grpcclient/gen"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
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
	t.Parallel()

	container, client := setupMinio(t)
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
	})

	uploader := NewUploader(client, &MockGRPC{}, UploaderConfig{
		Timeout: 3000,
		Bucket:  BucketName,
	})

	hello := []byte("hello, world!")
	helloHash := sha256.Sum256(hello)

	pngData := append([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, bytes.Repeat([]byte("a"), 100)...)
	pngHash := sha256.Sum256(pngData)

	zipData := append([]byte{0x50, 0x4B, 0x03, 0x04}, bytes.Repeat([]byte("z"), 100)...)
	zipHash := sha256.Sum256(zipData)

	var empty []byte
	emptyHash := sha256.Sum256(empty)

	testCases := []uploadIntegrationTestCase{
		{
			name:            "small valid file",
			content:         hello,
			fileSize:        int64(len(hello)),
			fileHash:        hex.EncodeToString(helloHash[:]),
			fileType:        "text/plain",
			expectError:     false,
			expectedSize:    int64(len(hello)),
			expectedType:    "text/plain",
			expectFinalName: true,
		},
		{
			name:            "image/png",
			content:         pngData,
			fileSize:        int64(len(pngData)),
			fileHash:        hex.EncodeToString(pngHash[:]),
			fileType:        "image/png",
			expectError:     false,
			expectedSize:    int64(len(pngData)),
			expectedType:    "image/png",
			expectFinalName: true,
		},
		{
			name:            "application/zip",
			content:         zipData,
			fileSize:        int64(len(zipData)),
			fileHash:        hex.EncodeToString(zipHash[:]),
			fileType:        "application/zip",
			expectError:     false,
			expectedSize:    int64(len(zipData)),
			expectedType:    "application/zip",
			expectFinalName: true,
		},
		{
			name:             "zero byte file",
			content:          empty,
			fileSize:         0,
			fileHash:         hex.EncodeToString(emptyHash[:]),
			fileType:         "text/plain",
			expectError:      true,
			expectedErrorMsg: "read error: empty file",
		},
		{
			name:             "hash mismatch",
			content:          hello,
			fileSize:         int64(len(hello)),
			fileHash:         strings.Repeat("0", 64),
			fileType:         "text/plain",
			expectError:      true,
			expectedErrorMsg: "invalid hash",
		},
		{
			name:             "file size mismatch",
			content:          hello,
			fileSize:         int64(len(hello)) + 5,
			fileHash:         hex.EncodeToString(helloHash[:]),
			fileType:         "text/plain",
			expectError:      true,
			expectedErrorMsg: "file size mismatch",
		},
		{
			name:             "simulate corrupted stream",
			content:          hello,
			fileSize:         int64(len(hello)),
			fileHash:         hex.EncodeToString(helloHash[:]),
			fileType:         "text/plain",
			simulateCorrupt:  true,
			expectError:      true,
			expectedErrorMsg: "failed to read file content",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

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

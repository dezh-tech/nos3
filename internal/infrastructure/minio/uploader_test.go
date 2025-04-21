package minio

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
)

var (
	testFileContent = []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
	} // "image/png"

	testFileSize = int64(len(testFileContent))
	testFileHash = func() string {
		h := sha256.Sum256(testFileContent)

		return hex.EncodeToString(h[:])
	}()
	testMimeType = "image/png"
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
		Creds:  credentials.NewStaticV4(TestAccessKey, TestSecretKey, ""),
		Secure: false,
	})
	if err != nil {
		t.Fatal("Failed to create minio client:", err)
	}

	for _, bucket := range []string{"images", "videos", "audios", "documents", "other"} {
		err = client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		if err != nil {
			t.Fatal("Failed to create bucket:", err)
		}
	}

	return container, client
}

func TestUploadFile(t *testing.T) {
	t.Parallel()

	container, client := setupMinio(t)
	t.Cleanup(func() {
		err := container.Terminate(context.Background())
		if err != nil {
			t.Fatal(err)
		}
	})

	uploader := NewUploader(client, 3000)

	tests := []struct {
		name             string
		content          []byte
		fileSize         int64
		fileType         string
		fileHash         string
		expectError      bool
		expectedErrorMsg string
		expectedSize     int64
		expectedType     string
	}{
		{
			name:         "successful upload",
			content:      testFileContent,
			fileSize:     testFileSize,
			fileType:     testMimeType,
			fileHash:     testFileHash,
			expectError:  false,
			expectedSize: testFileSize,
			expectedType: testMimeType,
		},
		{
			name:             "wrong file type",
			content:          testFileContent,
			fileSize:         testFileSize,
			fileType:         "application/json",
			fileHash:         testFileHash,
			expectError:      true,
			expectedErrorMsg: "invalid file type",
		},
		{
			name:             "wrong file size",
			content:          testFileContent,
			fileSize:         testFileSize + 1,
			fileType:         testMimeType,
			fileHash:         testFileHash,
			expectError:      true,
			expectedErrorMsg: "invalid file size",
		},
		{
			name:         "file size ignored (-1)",
			content:      testFileContent,
			fileSize:     -1,
			fileType:     testMimeType,
			fileHash:     testFileHash,
			expectError:  false,
			expectedSize: testFileSize,
			expectedType: testMimeType,
		},

		{
			name:         "file type ignored (empty)",
			content:      testFileContent,
			fileSize:     -1,
			fileType:     testMimeType,
			fileHash:     testFileHash,
			expectError:  false,
			expectedSize: testFileSize,
			expectedType: testMimeType,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			body := io.NopCloser(bytes.NewReader(tc.content))
			info, err := uploader.UploadFile(context.Background(), body, tc.fileSize, tc.fileHash, tc.fileType)

			if tc.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tc.expectedErrorMsg != "" && !strings.Contains(err.Error(), tc.expectedErrorMsg) {
					t.Errorf("error mismatch: expected to contain %q, got %q", tc.expectedErrorMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			} else {
				assert.Equal(t, info.Size, tc.expectedSize)
				assert.Equal(t, info.Type, tc.expectedType)
			}
		})
	}
}

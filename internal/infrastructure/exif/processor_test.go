package exif

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"nos3/internal/infrastructure/grpcclient/gen"
)

const (
	TestTimeout       = 5 * time.Second
	TestExifComment   = "Test EXIF comment with metadata"
	TestExifArtist    = "Test Artist"
	TestExifCopyright = "Copyright 2024"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "exif_processor_test_*")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})
	return dir
}

func checkExifToolAvailable(t *testing.T) {
	t.Helper()
	cmd := exec.Command("exiftool", "-ver")
	if err := cmd.Run(); err != nil {
		t.Skip("exiftool not available on this system, skipping integration test")
	}
}

func createTestImageWithExif(t *testing.T, filePath string) {
	t.Helper()
	jpegData := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01,
		0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x43,
		0x00, 0x08, 0x06, 0x06, 0x07, 0x06, 0x05, 0x08, 0x07, 0x07, 0x07, 0x09,
		0x09, 0x08, 0x0A, 0x0C, 0x14, 0x0D, 0x0C, 0x0B, 0x0B, 0x0C, 0x19, 0x12,
		0x13, 0x0F, 0x14, 0x1D, 0x1A, 0x1F, 0x1E, 0x1D, 0x1A, 0x1C, 0x1C, 0x20,
		0x24, 0x2E, 0x27, 0x20, 0x22, 0x2C, 0x23, 0x1C, 0x1C, 0x28, 0x37, 0x29,
		0x2C, 0x30, 0x31, 0x34, 0x34, 0x34, 0x1F, 0x27, 0x39, 0x3D, 0x38, 0x32,
		0x3C, 0x2E, 0x33, 0x34, 0x32, 0xFF, 0xC0, 0x00, 0x0B, 0x08, 0x00, 0x01,
		0x00, 0x01, 0x01, 0x01, 0x11, 0x00, 0xFF, 0xC4, 0x00, 0x14, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x03, 0xFF, 0xC4, 0x00, 0x14, 0x10, 0x01, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0xFF, 0xDA, 0x00, 0x08, 0x01, 0x01, 0x00, 0x00, 0x3F, 0x00,
		0x37, 0xFF, 0xD9,
	}
	err := os.WriteFile(filePath, jpegData, 0644)
	require.NoError(t, err)
	cmd := exec.Command("exiftool",
		"-overwrite_original",
		fmt.Sprintf("-Comment=%s", TestExifComment),
		fmt.Sprintf("-Artist=%s", TestExifArtist),
		fmt.Sprintf("-Copyright=%s", TestExifCopyright),
		"-Make=TestCamera",
		"-Model=TestModel",
		"-Software=TestSoftware",
		filePath,
	)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to add EXIF data: %s", string(output))
}

func createTestPNGWithExif(t *testing.T, filePath string) {
	t.Helper()
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, 0x89,
		0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41, 0x54,
		0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00, 0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4,
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44,
		0xAE, 0x42, 0x60, 0x82,
	}
	err := os.WriteFile(filePath, pngData, 0644)
	require.NoError(t, err)
	cmd := exec.Command("exiftool",
		"-overwrite_original",
		fmt.Sprintf("-Comment=%s", TestExifComment),
		fmt.Sprintf("-Artist=%s", TestExifArtist),
		filePath,
	)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Failed to add metadata: %s", string(output))
}

func hasExifData(t *testing.T, filePath string) bool {
	t.Helper()
	cmd := exec.Command("exiftool", "-a", "-G1", filePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	outputStr := string(output)
	return strings.Contains(outputStr, TestExifComment) ||
		strings.Contains(outputStr, TestExifArtist) ||
		strings.Contains(outputStr, TestExifCopyright) ||
		strings.Contains(outputStr, "TestCamera")
}

func TestProcessor_RemoveExifData_JPEG(t *testing.T) {
	checkExifToolAvailable(t)

	testDir := setupTestDir(t)
	testFile := filepath.Join(testDir, "test_image.jpg")
	createTestImageWithExif(t, testFile)

	require.True(t, hasExifData(t, testFile), "Test image should have EXIF data")

	mockGRPC := &MockGRPC{}
	processor, err := NewProcessor("", TestTimeout, mockGRPC)
	require.NoError(t, err)
	defer processor.Close()

	err = processor.RemoveExifData(context.Background(), testFile)
	assert.NoError(t, err)

	assert.False(t, hasExifData(t, testFile), "EXIF data should be removed")
	fileInfo, err := os.Stat(testFile)
	assert.NoError(t, err)
	assert.Greater(t, fileInfo.Size(), int64(0), "File should not be empty")
}

func TestProcessor_RemoveExifData_PNG(t *testing.T) {
	checkExifToolAvailable(t)

	testDir := setupTestDir(t)
	testFile := filepath.Join(testDir, "test_image.png")
	createTestPNGWithExif(t, testFile)

	require.True(t, hasExifData(t, testFile), "Test PNG should have metadata")

	mockGRPC := &MockGRPC{}
	processor, err := NewProcessor("", TestTimeout, mockGRPC)
	require.NoError(t, err)
	defer processor.Close()

	err = processor.RemoveExifData(context.Background(), testFile)
	assert.NoError(t, err)

	assert.False(t, hasExifData(t, testFile), "Metadata should be removed")
	fileInfo, err := os.Stat(testFile)
	assert.NoError(t, err)
	assert.Greater(t, fileInfo.Size(), int64(0), "File should not be empty")
}

func TestProcessor_RemoveExifData_NonExistentFile(t *testing.T) {
	checkExifToolAvailable(t)

	testDir := setupTestDir(t)
	nonExistentFile := filepath.Join(testDir, "does_not_exist.jpg")

	mockGRPC := &MockGRPC{}
	processor, err := NewProcessor("", TestTimeout, mockGRPC)
	require.NoError(t, err)
	defer processor.Close()

	err = processor.RemoveExifData(context.Background(), nonExistentFile)
	if err != nil {
		var exifErr *ExifError
		if assert.ErrorAs(t, err, &exifErr) {
			assert.Equal(t, ErrorCodeExifRemovalFailed, exifErr.Code)
		}
	}
}

func TestProcessor_RemoveExifData_ContextCancellation(t *testing.T) {
	checkExifToolAvailable(t)

	testDir := setupTestDir(t)
	testFile := filepath.Join(testDir, "test_image.jpg")
	createTestImageWithExif(t, testFile)

	mockGRPC := &MockGRPC{}
	processor, err := NewProcessor("", TestTimeout, mockGRPC)
	require.NoError(t, err)
	defer processor.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = processor.RemoveExifData(ctx, testFile)
	assert.Error(t, err)

	var exifErr *ExifError
	if assert.ErrorAs(t, err, &exifErr) {
		assert.Equal(t, ErrorCodeTimeout, exifErr.Code)
		assert.True(t, exifErr.IsTimeout())
	}
}

func TestProcessor_Close(t *testing.T) {
	checkExifToolAvailable(t)

	mockGRPC := &MockGRPC{}
	processor, err := NewProcessor("", TestTimeout, mockGRPC)
	require.NoError(t, err)

	err = processor.Close()
	assert.NoError(t, err)
}

func TestProcessor_NewProcessor_InvalidExifToolPath(t *testing.T) {
	mockGRPC := &MockGRPC{}
	mockGRPC.On("AddLog", mock.Anything, mock.Anything).Return(&gen.AddLogResponse{}, nil).Maybe()

	processor, err := NewProcessor("/invalid/path/to/exiftool", TestTimeout, mockGRPC)
	assert.Error(t, err)
	assert.Nil(t, processor)

	var exifErr *ExifError
	if assert.ErrorAs(t, err, &exifErr) {
		assert.Equal(t, ErrorCodeExifToolNotFound, exifErr.Code)
	}
}

package exif

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	RemoverTimeout = 5 * time.Second
)

func TestNewRemover(t *testing.T) {
	t.Parallel()

	mockValidator := &MockFileValidator{}
	mockProcessor := &MockProcessor{}
	mockGRPC := &MockGRPC{}

	remover := NewRemover(mockValidator, mockProcessor, RemoverTimeout, mockGRPC)

	require.NotNil(t, remover)
	assert.Equal(t, RemoverTimeout, remover.timeout)
	assert.Equal(t, mockValidator, remover.fileValidator)
	assert.Equal(t, mockProcessor, remover.exifProcessor)
	assert.Equal(t, mockGRPC, remover.grpcClient)
}

func TestRemoveExifFromFile_Success(t *testing.T) {
	t.Parallel()

	mockValidator := &MockFileValidator{}
	mockProcessor := &MockProcessor{}
	mockGRPC := &MockGRPC{}

	mockValidator.On("ValidateFileType", TestFilePath).Return(nil)
	mockProcessor.On("RemoveExifData", mock.Anything, TestFilePath).Return(nil)

	remover := NewRemover(mockValidator, mockProcessor, RemoverTimeout, mockGRPC)

	err := remover.RemoveExifFromFile(context.Background(), TestFilePath)

	require.NoError(t, err)
	mockValidator.AssertExpectations(t)
	mockProcessor.AssertExpectations(t)
}

func TestRemoveExifFromFile_UnsupportedFileType(t *testing.T) {
	t.Parallel()

	mockValidator := &MockFileValidator{}
	mockProcessor := &MockProcessor{}
	mockGRPC := &MockGRPC{}

	unsupportedErr := &Error{
		Code:    ErrorCodeUnsupportedFileType,
		Message: "file type does not support EXIF data",
		Details: ".txt not supported",
	}

	mockValidator.On("ValidateFileType", TestUnsupportedPath).Return(unsupportedErr)

	remover := NewRemover(mockValidator, mockProcessor, RemoverTimeout, mockGRPC)

	err := remover.RemoveExifFromFile(context.Background(), TestUnsupportedPath)

	require.NoError(t, err)
	mockValidator.AssertExpectations(t)
	mockProcessor.AssertNotCalled(t, "RemoveExifData", mock.Anything, mock.Anything)
}

func TestRemoveExifFromFile_ValidationError(t *testing.T) {
	t.Parallel()

	mockValidator := &MockFileValidator{}
	mockProcessor := &MockProcessor{}
	mockGRPC := &MockGRPC{}

	validationErr := &Error{
		Code:    ErrorCodeExifToolNotFound,
		Message: "failed to get supported extensions",
		Details: "exiftool not found",
	}

	mockValidator.On("ValidateFileType", TestFilePath).Return(validationErr)

	remover := NewRemover(mockValidator, mockProcessor, RemoverTimeout, mockGRPC)

	err := remover.RemoveExifFromFile(context.Background(), TestFilePath)

	require.Error(t, err)
	assert.Equal(t, validationErr, err)
	mockValidator.AssertExpectations(t)
	mockProcessor.AssertNotCalled(t, "RemoveExifData", mock.Anything, mock.Anything)
}

func TestRemoveExifFromFile_ProcessorError(t *testing.T) {
	t.Parallel()

	mockValidator := &MockFileValidator{}
	mockProcessor := &MockProcessor{}
	mockGRPC := &MockGRPC{}

	processorErr := &Error{
		Code:    ErrorCodeExifRemovalFailed,
		Message: "failed to remove EXIF data",
		Details: "write error",
	}

	mockValidator.On("ValidateFileType", TestCorruptedPath).Return(nil)
	mockProcessor.On("RemoveExifData", mock.Anything, TestCorruptedPath).Return(processorErr)

	remover := NewRemover(mockValidator, mockProcessor, RemoverTimeout, mockGRPC)

	err := remover.RemoveExifFromFile(context.Background(), TestCorruptedPath)

	require.Error(t, err)
	assert.Equal(t, processorErr, err)
	mockValidator.AssertExpectations(t)
	mockProcessor.AssertExpectations(t)
}

func TestRemoveExifFromFile_ContextCancellation(t *testing.T) {
	t.Parallel()

	mockValidator := &MockFileValidator{}
	mockProcessor := &MockProcessor{}
	mockGRPC := &MockGRPC{}

	mockValidator.On("ValidateFileType", TestFilePath).Return(nil)
	mockProcessor.On("RemoveExifData", mock.Anything, TestFilePath).
		Return(&Error{
			Code:    ErrorCodeTimeout,
			Message: "context cancelled",
			Details: "operation cancelled",
		})

	remover := NewRemover(mockValidator, mockProcessor, RemoverTimeout, mockGRPC)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := remover.RemoveExifFromFile(ctx, TestFilePath)

	require.Error(t, err)
	mockValidator.AssertExpectations(t)
}

func TestRemoveExifFromFile_MultipleFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		filePath     string
		validateErr  error
		processErr   error
		expectError  bool
		expectedCode ErrorCode
	}{
		{
			name:        "successful removal",
			filePath:    "/images/photo1.jpg",
			validateErr: nil,
			processErr:  nil,
			expectError: false,
		},
		{
			name:        "unsupported file",
			filePath:    "/docs/readme.txt",
			validateErr: &Error{Code: ErrorCodeUnsupportedFileType},
			processErr:  nil,
			expectError: false,
		},
		{
			name:         "validation fails",
			filePath:     "/images/photo2.jpg",
			validateErr:  &Error{Code: ErrorCodeExifToolNotFound},
			processErr:   nil,
			expectError:  true,
			expectedCode: ErrorCodeExifToolNotFound,
		},
		{
			name:         "processing fails",
			filePath:     "/images/photo3.jpg",
			validateErr:  nil,
			processErr:   &Error{Code: ErrorCodeExifRemovalFailed},
			expectError:  true,
			expectedCode: ErrorCodeExifRemovalFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockValidator := &MockFileValidator{}
			mockProcessor := &MockProcessor{}
			mockGRPC := &MockGRPC{}

			mockValidator.On("ValidateFileType", tt.filePath).Return(tt.validateErr)

			if tt.validateErr == nil {
				mockProcessor.On("RemoveExifData", mock.Anything, tt.filePath).Return(tt.processErr)
			}

			remover := NewRemover(mockValidator, mockProcessor, RemoverTimeout, mockGRPC)

			err := remover.RemoveExifFromFile(context.Background(), tt.filePath)

			if tt.expectError {
				require.Error(t, err)
				var exifErr *Error
				require.True(t, errors.As(err, &exifErr))
				assert.Equal(t, tt.expectedCode, exifErr.Code)
			} else {
				require.NoError(t, err)
			}

			mockValidator.AssertExpectations(t)
			mockProcessor.AssertExpectations(t)
		})
	}
}

func TestClose(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		closeErr    error
		expectError bool
	}{
		{
			name:        "successful close",
			closeErr:    nil,
			expectError: false,
		},
		{
			name:        "close error",
			closeErr:    errors.New("failed to close exiftool"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockValidator := &MockFileValidator{}
			mockProcessor := &MockProcessor{}
			mockGRPC := &MockGRPC{}

			mockProcessor.On("Close").Return(tt.closeErr)

			remover := NewRemover(mockValidator, mockProcessor, RemoverTimeout, mockGRPC)

			err := remover.Close()

			if tt.expectError {
				require.Error(t, err)
				assert.Equal(t, tt.closeErr, err)
			} else {
				require.NoError(t, err)
			}

			mockProcessor.AssertExpectations(t)
		})
	}
}

package exif

import (
	"context"

	"github.com/stretchr/testify/mock"

	"nos3/internal/infrastructure/grpcclient/gen"
)

// MockGRPC is a shared mock implementation of the gRPC client interface
type MockGRPC struct {
	mock.Mock
}

func (m *MockGRPC) RegisterService(_ context.Context, _, _ string) (*gen.RegisterServiceResponse, error) {
	args := m.Called()
	return args.Get(0).(*gen.RegisterServiceResponse), args.Error(1)
}

func (m *MockGRPC) AddLog(_ context.Context, msg, stack string) (*gen.AddLogResponse, error) {
	args := m.Called(msg, stack)
	return args.Get(0).(*gen.AddLogResponse), args.Error(1)
}

func (m *MockGRPC) AddReport(_ context.Context, _ string, _ []string, _, _, _, _ string) (
	*gen.AddReportResponse, error,
) {
	args := m.Called()
	return args.Get(0).(*gen.AddReportResponse), args.Error(1)
}

// MockCommandExecutor is a mock for command execution
type MockCommandExecutor struct {
	mock.Mock
}

func (m *MockCommandExecutor) Execute(cmd string, args ...string) ([]byte, error) {
	argsList := m.Called(cmd, args)
	if argsList.Get(0) == nil {
		return nil, argsList.Error(1)
	}
	return argsList.Get(0).([]byte), argsList.Error(1)
}

// MockExtensionProvider is a mock for extension provider
type MockExtensionProvider struct {
	mock.Mock
}

func (m *MockExtensionProvider) GetSupportedExtensions() (map[string]bool, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]bool), args.Error(1)
}

// MockExifProcessor is a mock for exif processor
type MockExifProcessor struct {
	mock.Mock
}

func (m *MockExifProcessor) RemoveExifData(ctx context.Context, filePath string) error {
	args := m.Called(ctx, filePath)
	return args.Error(0)
}

func (m *MockExifProcessor) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockFileValidator is a mock for file validator
type MockFileValidator struct {
	mock.Mock
}

func (m *MockFileValidator) ValidateFileType(filePath string) error {
	args := m.Called(filePath)
	return args.Error(0)
}

const (
	TestImagePath       = "/tmp/test_image.jpg"
	TestVideoPath       = "/tmp/test_video.mp4"
	TestFilePath        = "/test/image.jpg"
	TestUnsupportedPath = "/test/document.txt"
	TestCorruptedPath   = "/test/corrupted.jpg"
	TestTimeoutPath     = "/test/timeout.jpg"
	TestExifToolPath    = "/usr/bin/exiftool"
	InvalidExifToolPath = "/invalid/path/exiftool"
)

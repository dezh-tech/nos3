package clamav

import (
	"bytes"
	"context"
	"fmt"
	"github.com/dutchcoders/go-clamd"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"nos3/internal/domain/entity"
	"nos3/internal/infrastructure/grpcclient/gen"
)

const (
	ClamAVImage = "clamav/clamav:latest"
)

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
func setupClamAV(t *testing.T) (string, func()) {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        ClamAVImage,
		ExposedPorts: []string{"3310/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("3310/tcp").WithStartupTimeout(60 * time.Second),
		),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatal("Failed to start ClamAV container:", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal("Failed to get container host:", err)
	}

	port, err := container.MappedPort(ctx, "3310")
	if err != nil {
		t.Fatal("Failed to get mapped port:", err)
	}

	address := fmt.Sprintf("tcp://%s", net.JoinHostPort(host, port.Port()))

	return address, func() {
		_ = container.Terminate(ctx)
	}
}
func TestScanStream_CleanFile(t *testing.T) {
	// t.Parallel()

	address, cleanup := setupClamAV(t)
	t.Cleanup(cleanup)

	mockGRPC := &MockGRPC{}
	mockGRPC.On("AddLog", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(&gen.AddLogResponse{}, nil).Maybe()

	scanner, err := NewScanner(ScannerConfig{
		Address: address,
		Timeout: 30000,
	}, mockGRPC)
	require.NoError(t, err)

	cleanContent := "This is a clean test file with no malware."
	reader := strings.NewReader(cleanContent)

	result, err := scanner.ScanStream(context.Background(), reader)

	assert.NoError(t, err)
	assert.Equal(t, entity.MalwareScanStatusClean, result.Status)
	assert.True(t, result.IsClean())
	assert.Equal(t, 0, result.ThreatCount())
}

func TestScanStream_InfectedFile(t *testing.T) {
	// t.Parallel()

	address, cleanup := setupClamAV(t)
	t.Cleanup(cleanup)

	mockGRPC := &MockGRPC{}
	mockGRPC.On("AddLog", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(&gen.AddLogResponse{}, nil).Maybe()

	scanner, err := NewScanner(ScannerConfig{
		Address: address,
		Timeout: 30000,
	}, mockGRPC)
	require.NoError(t, err)

	reader := bytes.NewReader(clamd.EICAR)

	result, err := scanner.ScanStream(context.Background(), reader)

	assert.NoError(t, err)
	assert.Equal(t, entity.MalwareScanStatusInfected, result.Status)
	assert.NotEmpty(t, result.Threats)
	assert.Contains(t, strings.ToUpper(result.Threats[0]), "EICAR")
}
func TestScanStream_EmptyFile(t *testing.T) {
	// t.Parallel()

	address, cleanup := setupClamAV(t)
	t.Cleanup(cleanup)

	mockGRPC := &MockGRPC{}
	mockGRPC.On("AddLog", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(&gen.AddLogResponse{}, nil).Maybe()

	scanner, err := NewScanner(ScannerConfig{
		Address: address,
		Timeout: 30000,
	}, mockGRPC)
	require.NoError(t, err)

	reader := strings.NewReader("")

	result, err := scanner.ScanStream(context.Background(), reader)

	assert.NoError(t, err)
	assert.Equal(t, entity.MalwareScanStatusClean, result.Status)
	assert.True(t, result.IsClean())
	assert.False(t, result.HasError())
	assert.Equal(t, 0, result.ThreatCount())
}

func TestScanStream_LargeCleanFile(t *testing.T) {
	// t.Parallel()

	address, cleanup := setupClamAV(t)
	t.Cleanup(cleanup)

	mockGRPC := &MockGRPC{}
	mockGRPC.On("AddLog", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(&gen.AddLogResponse{}, nil).Maybe()

	scanner, err := NewScanner(ScannerConfig{
		Address: address,
		Timeout: 30000,
	}, mockGRPC)
	require.NoError(t, err)

	largeContent := strings.Repeat("This is a clean file content. ", 350000)
	reader := strings.NewReader(largeContent)

	result, err := scanner.ScanStream(context.Background(), reader)

	assert.NoError(t, err)
	assert.Equal(t, entity.MalwareScanStatusClean, result.Status)
	assert.True(t, result.IsClean())
	assert.False(t, result.HasError())
	assert.Equal(t, 0, result.ThreatCount())
}

func TestScanStream_BinaryCleanFile(t *testing.T) {
	// t.Parallel()

	address, cleanup := setupClamAV(t)
	t.Cleanup(cleanup)

	mockGRPC := &MockGRPC{}
	mockGRPC.On("AddLog", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(&gen.AddLogResponse{}, nil).Maybe()

	scanner, err := NewScanner(ScannerConfig{
		Address: address,
		Timeout: 30000,
	}, mockGRPC)
	require.NoError(t, err)

	binaryContent := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	binaryContent = append(binaryContent, bytes.Repeat([]byte{0x00, 0x01, 0x02, 0x03}, 1000)...)
	reader := bytes.NewReader(binaryContent)

	result, err := scanner.ScanStream(context.Background(), reader)

	assert.NoError(t, err)
	assert.Equal(t, entity.MalwareScanStatusClean, result.Status)
	assert.True(t, result.IsClean())
	assert.False(t, result.HasError())
	assert.Equal(t, 0, result.ThreatCount())
}

func setupClamAVBenchmark(b *testing.B) (string, func()) {
	b.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        ClamAVImage,
		ExposedPorts: []string{"3310/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("3310/tcp").WithStartupTimeout(60 * time.Second),
		),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		b.Fatal("Failed to start ClamAV container:", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		b.Fatal("Failed to get container host:", err)
	}

	port, err := container.MappedPort(ctx, "3310")
	if err != nil {
		b.Fatal("Failed to get mapped port:", err)
	}

	address := fmt.Sprintf("tcp://%s", net.JoinHostPort(host, port.Port()))

	return address, func() {
		_ = container.Terminate(ctx)
	}
}

func BenchmarkScanStream_SmallCleanFile(b *testing.B) {
	address, cleanup := setupClamAVBenchmark(b)
	defer cleanup()

	mockGRPC := &MockGRPC{}
	mockGRPC.On("AddLog", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(&gen.AddLogResponse{}, nil).Maybe()

	scanner, err := NewScanner(ScannerConfig{
		Address: address,
		Timeout: 30000,
	}, mockGRPC)
	if err != nil {
		b.Fatal("Failed to create scanner:", err)
	}

	content := "This is a small clean test file for benchmarking."

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(content)
		_, err := scanner.ScanStream(context.Background(), reader)
		if err != nil {
			b.Fatal("Scan failed:", err)
		}
	}
}

func BenchmarkScanStream_LargeCleanFile(b *testing.B) {
	address, cleanup := setupClamAVBenchmark(b)
	defer cleanup()

	mockGRPC := &MockGRPC{}
	mockGRPC.On("AddLog", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(&gen.AddLogResponse{}, nil).Maybe()

	scanner, err := NewScanner(ScannerConfig{
		Address: address,
		Timeout: 30000,
	}, mockGRPC)
	if err != nil {
		b.Fatal("Failed to create scanner:", err)
	}

	content := strings.Repeat("Large file content for benchmarking. ", 30000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := strings.NewReader(content)
		_, err := scanner.ScanStream(context.Background(), reader)
		if err != nil {
			b.Fatal("Scan failed:", err)
		}
	}
}

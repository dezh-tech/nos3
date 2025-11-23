package clamav

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/swimmingrieux/go-clamd"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"nos3/internal/domain/entity"
	"nos3/internal/domain/repository/clamav"
	"nos3/internal/infrastructure/grpcclient/gen"
)

const (
	ClamAVImage                  = "clamav/clamav:latest"
	ClamAVPort                   = "3310/tcp"
	ClamAVAddressFormat          = "tcp://%s"
	CleanFileContent             = "This is a clean test file with no malware."
	EmptyFileContent             = ""
	LargeCleanFileContent        = "This is a clean file content. "
	InfectedContent              = "infected content"
	UnparseableContent           = "unparseable content"
	SomeContent                  = "some content"
	MalwareDescTrojan            = "Trojan.Generic.123"
	MalwareDescVirus             = "Virus.Win32.Test"
	MalwareDescMalwareSuspicious = "Malware.Suspicious.456"
	ScanErrorCorruptedData       = "Unable to scan file: corrupted data"
	ParseErrorFileFormatNotRecog = "File format not recognized"
	MalwareDescTrojanTest        = "Trojan.Test.123"
	MalwareDescVirusTest         = "Virus.Test.456"
	Timeout                      = 30000
	StartupTimeoutDuration       = 60 * time.Second
	LargeFileContentRepeatCount  = 350000
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

type MockClamdClient struct {
	mock.Mock
}

func (m *MockClamdClient) Ping() error {
	args := m.Called()

	return args.Error(0)
}

func (m *MockClamdClient) ScanStream(reader io.Reader, abort chan bool) (chan *clamd.ScanResult, error) {
	args := m.Called(reader, abort)

	return args.Get(0).(chan *clamd.ScanResult), args.Error(1)
}

func createMockScanner(clamdClient clamav.ClamdClient, grpcClient *MockGRPC) *Scanner {
	return &Scanner{
		clamd:      clamdClient,
		timeout:    30 * time.Second,
		grpcClient: grpcClient,
	}
}

func createMockResultChannel(results []*clamd.ScanResult) chan *clamd.ScanResult {
	resultChan := make(chan *clamd.ScanResult, len(results))
	for _, result := range results {
		resultChan <- result
	}
	close(resultChan)

	return resultChan
}

func setupClamAV(t *testing.T) (string, func()) {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        ClamAVImage,
		ExposedPorts: []string{ClamAVPort},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort(ClamAVPort).WithStartupTimeout(StartupTimeoutDuration),
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

	address := fmt.Sprintf(ClamAVAddressFormat, net.JoinHostPort(host, port.Port()))

	waitForClamAVReady(t, address)

	return address, func() {
		_ = container.Terminate(ctx)
	}
}

func waitForClamAVReady(t *testing.T, address string) {
	t.Helper()
	maxRetries := 30
	retryDelay := 2 * time.Second

	mockGRPC := &MockGRPC{}
	mockGRPC.On("AddLog", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(&gen.AddLogResponse{}, nil)

	for i := 0; i < maxRetries; i++ {
		scanner, err := NewScanner(ScannerConfig{
			Address: address,
			Timeout: Timeout,
		}, mockGRPC)

		if err == nil && scanner != nil {
			// Successfully connected and pinged
			return
		}

		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}

	t.Fatal("ClamAV daemon did not become ready within timeout period")
}

func TestScanStream_CleanFile(t *testing.T) {
	// t.Parallel()

	address, cleanup := setupClamAV(t)
	t.Cleanup(cleanup)

	mockGRPC := &MockGRPC{}
	mockGRPC.On("AddLog", mock.AnythingOfType("string"), mock.AnythingOfType("string")).
		Return(&gen.AddLogResponse{}, nil)

	scanner, err := NewScanner(ScannerConfig{
		Address: address,
		Timeout: Timeout,
	}, mockGRPC)
	require.NoError(t, err)

	cleanContent := CleanFileContent
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
		Return(&gen.AddLogResponse{}, nil)

	scanner, err := NewScanner(ScannerConfig{
		Address: address,
		Timeout: Timeout,
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
		Return(&gen.AddLogResponse{}, nil)

	scanner, err := NewScanner(ScannerConfig{
		Address: address,
		Timeout: Timeout,
	}, mockGRPC)
	require.NoError(t, err)

	reader := strings.NewReader(EmptyFileContent)

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
		Return(&gen.AddLogResponse{}, nil)

	scanner, err := NewScanner(ScannerConfig{
		Address: address,
		Timeout: Timeout,
	}, mockGRPC)
	require.NoError(t, err)

	largeContent := strings.Repeat(LargeCleanFileContent, LargeFileContentRepeatCount)
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
		Return(&gen.AddLogResponse{}, nil)

	scanner, err := NewScanner(ScannerConfig{
		Address: address,
		Timeout: Timeout,
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

func TestScanStream_MultipleThreatsDetected(t *testing.T) {
	mockGRPC := &MockGRPC{}
	mockClamd := &MockClamdClient{}

	results := []*clamd.ScanResult{
		{Status: clamd.RES_FOUND, Description: MalwareDescTrojan},
		{Status: clamd.RES_FOUND, Description: MalwareDescVirus},
		{Status: clamd.RES_FOUND, Description: MalwareDescMalwareSuspicious},
	}
	resultChan := createMockResultChannel(results)

	mockClamd.On("ScanStream", mock.Anything, mock.Anything).Return(resultChan, nil)

	scanner := createMockScanner(mockClamd, mockGRPC)
	reader := strings.NewReader(InfectedContent)

	result, err := scanner.ScanStream(context.Background(), reader)

	assert.NoError(t, err)
	assert.Equal(t, entity.MalwareScanStatusInfected, result.Status)
	assert.Equal(t, 3, len(result.Threats))
	assert.Contains(t, result.Threats, MalwareDescTrojan)
	assert.Contains(t, result.Threats, MalwareDescVirus)
	assert.Contains(t, result.Threats, MalwareDescMalwareSuspicious)
}

func TestScanStream_ScanErrorDuringScan(t *testing.T) {
	mockGRPC := &MockGRPC{}
	mockClamd := &MockClamdClient{}

	results := []*clamd.ScanResult{
		{Status: clamd.RES_ERROR, Description: ScanErrorCorruptedData},
	}
	resultChan := createMockResultChannel(results)

	mockClamd.On("ScanStream", mock.Anything, mock.Anything).Return(resultChan, nil)

	scanner := createMockScanner(mockClamd, mockGRPC)
	reader := strings.NewReader(SomeContent)

	result, err := scanner.ScanStream(context.Background(), reader)

	assert.Error(t, err)
	assert.Equal(t, entity.MalwareScanStatusError, result.Status)
	assert.Contains(t, result.Error, "scan failed")

	var malwareErr *MalwareError
	ok := errors.As(err, &malwareErr)

	assert.True(t, ok)
	assert.Equal(t, ErrorCodeScanFailed, malwareErr.Code)
	assert.Equal(t, ScanErrorCorruptedData, malwareErr.Details)
}

func TestScanStream_MixedResults(t *testing.T) {
	mockGRPC := &MockGRPC{}
	mockClamd := &MockClamdClient{}

	results := []*clamd.ScanResult{
		{Status: clamd.RES_OK, Description: ""},
		{Status: clamd.RES_FOUND, Description: MalwareDescTrojanTest},
		{Status: clamd.RES_OK, Description: ""},
		{Status: clamd.RES_FOUND, Description: MalwareDescVirusTest},
		{Status: clamd.RES_OK, Description: ""},
	}
	resultChan := createMockResultChannel(results)

	mockClamd.On("ScanStream", mock.Anything, mock.Anything).Return(resultChan, nil)

	scanner := createMockScanner(mockClamd, mockGRPC)
	reader := strings.NewReader("mixed content")

	result, err := scanner.ScanStream(context.Background(), reader)

	assert.NoError(t, err)
	assert.Equal(t, entity.MalwareScanStatusInfected, result.Status)
	assert.Equal(t, 2, len(result.Threats))
	assert.Contains(t, result.Threats, MalwareDescTrojanTest)
	assert.Contains(t, result.Threats, MalwareDescVirusTest)
}

func TestScanStream_ParseErrorDuringScan(t *testing.T) {
	mockGRPC := &MockGRPC{}
	mockClamd := &MockClamdClient{}

	results := []*clamd.ScanResult{
		{Status: clamd.RES_PARSE_ERROR, Description: ParseErrorFileFormatNotRecog},
	}
	resultChan := createMockResultChannel(results)

	mockClamd.On("ScanStream", mock.Anything, mock.Anything).Return(resultChan, nil)

	scanner := createMockScanner(mockClamd, mockGRPC)
	reader := strings.NewReader(UnparseableContent)

	result, err := scanner.ScanStream(context.Background(), reader)

	assert.Error(t, err)
	assert.Equal(t, entity.MalwareScanStatusError, result.Status)
	assert.Contains(t, result.Error, "scan failed")

	var malwareErr *MalwareError
	ok := errors.As(err, &malwareErr)

	assert.True(t, ok)
	assert.Equal(t, ErrorCodeScanFailed, malwareErr.Code)
	assert.Equal(t, ParseErrorFileFormatNotRecog, malwareErr.Details)
}

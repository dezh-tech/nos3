package exif

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	DefaultExifToolCmd   = ""
	CustomExifToolCmd    = "/usr/bin/exiftool"
	InvalidExifToolCmd   = "invalid-exiftool-path"
	DefaultListFlag      = "-listf"
	SampleOutput         = `JPG JPEG PNG GIF TIFF TIF BMP RAW CR2 NEF ARW DNG MP4 MOV AVI PDF`
	MinimalOutput        = `JPG PNG`
	ThreeExtensionOutput = `JPG PNG GIF`
	MixedCaseOutput      = `JPG Jpeg PnG gif`
	EmptyOutput          = ``
)

func createTestConfig(exiftoolCmd, listFlag string) ExtensionProviderConfig {
	return ExtensionProviderConfig{
		ExifToolCmd:      exiftoolCmd,
		ExifToolListFlag: listFlag,
	}
}

func TestGetSupportedExtensions_Success(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(DefaultExifToolCmd, DefaultListFlag)
	mockExecutor := &MockCommandExecutor{}
	mockExecutor.On("Execute", DefaultExifToolCmd, []string{DefaultListFlag}).Return([]byte(SampleOutput), nil)

	provider := NewExtensionProvider(cfg, mockExecutor)

	extensions, err := provider.GetSupportedExtensions()

	require.NoError(t, err)
	require.NotNil(t, extensions)
	assert.True(t, extensions[".jpg"])
	assert.True(t, extensions[".png"])
	assert.True(t, extensions[".mp4"])
	assert.False(t, extensions[".txt"])
	mockExecutor.AssertExpectations(t)
}

func TestGetSupportedExtensions_CachingBehavior(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(DefaultExifToolCmd, DefaultListFlag)
	mockExecutor := &MockCommandExecutor{}
	mockExecutor.On("Execute", DefaultExifToolCmd, []string{DefaultListFlag}).Return([]byte(MinimalOutput), nil).Once()

	provider := NewExtensionProvider(cfg, mockExecutor)

	ext1, err1 := provider.GetSupportedExtensions()
	require.NoError(t, err1)
	require.Equal(t, 2, len(ext1))

	ext1[".gif"] = true

	ext2, err2 := provider.GetSupportedExtensions()
	require.NoError(t, err2)
	assert.Equal(t, 2, len(ext2))
	assert.False(t, ext2[".gif"])

	mockExecutor.AssertExpectations(t)
}

func TestGetSupportedExtensions_ErrorHandling(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(InvalidExifToolCmd, DefaultListFlag)
	mockExecutor := &MockCommandExecutor{}
	mockExecutor.On("Execute", InvalidExifToolCmd, []string{DefaultListFlag}).Return(nil, errors.New("command not found"))

	provider := NewExtensionProvider(cfg, mockExecutor)

	extensions, err := provider.GetSupportedExtensions()

	require.Error(t, err)
	assert.Nil(t, extensions)
	mockExecutor.AssertExpectations(t)
}

func TestGetSupportedExtensions_EmptyExtensions(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(DefaultExifToolCmd, DefaultListFlag)
	mockExecutor := &MockCommandExecutor{}
	mockExecutor.On("Execute", DefaultExifToolCmd, []string{DefaultListFlag}).Return([]byte(EmptyOutput), nil)

	provider := NewExtensionProvider(cfg, mockExecutor)

	extensions, err := provider.GetSupportedExtensions()

	require.NoError(t, err)
	require.NotNil(t, extensions)
	assert.Equal(t, 0, len(extensions))
	mockExecutor.AssertExpectations(t)
}

func TestFetchSupportedExtensions_ParseOutput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		output         string
		expectedCount  int
		shouldContain  []string
		shouldNotExist []string
	}{
		{
			name: "standard output",
			output: `Recognized file extensions:
  JPG JPEG PNG GIF
  MP4 MOV AVI
  PDF DOC DOCX`,
			expectedCount:  10,
			shouldContain:  []string{".jpg", ".jpeg", ".png", ".gif", ".mp4", ".pdf"},
			shouldNotExist: []string{".txt", ".exe"},
		},
		{
			name:           "single line",
			output:         `JPG PNG GIF`,
			expectedCount:  3,
			shouldContain:  []string{".jpg", ".png", ".gif"},
			shouldNotExist: []string{".mp4"},
		},
		{
			name: "with headers and data lines",
			output: `Recognized file extensions:
  JPG PNG
  MP4 AVI`,
			expectedCount:  4,
			shouldContain:  []string{".jpg", ".png", ".mp4", ".avi"},
			shouldNotExist: []string{".txt"},
		},
		{
			name:           "empty output",
			output:         "",
			expectedCount:  0,
			shouldContain:  []string{},
			shouldNotExist: []string{".jpg"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := createTestConfig(DefaultExifToolCmd, DefaultListFlag)
			mockExecutor := &MockCommandExecutor{}
			mockExecutor.On("Execute", DefaultExifToolCmd, []string{DefaultListFlag}).Return([]byte(tt.output), nil)

			provider := NewExtensionProvider(cfg, mockExecutor)

			result, err := provider.GetSupportedExtensions()
			require.NoError(t, err)
			require.Equal(t, tt.expectedCount, len(result))

			for _, ext := range tt.shouldContain {
				assert.True(t, result[ext], "should contain %s", ext)
			}

			for _, ext := range tt.shouldNotExist {
				assert.False(t, result[ext], "should not contain %s", ext)
			}

			mockExecutor.AssertExpectations(t)
		})
	}
}

func TestNewExtensionProvider(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		exiftoolCmd string
		listFlag    string
	}{
		{"with custom command", CustomExifToolCmd, DefaultListFlag},
		{"with default command", DefaultExifToolCmd, DefaultListFlag},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := createTestConfig(tt.exiftoolCmd, tt.listFlag)
			mockExecutor := &MockCommandExecutor{}
			provider := NewExtensionProvider(cfg, mockExecutor)

			require.NotNil(t, provider)
			assert.Equal(t, tt.exiftoolCmd, provider.exiftoolCmd)
			assert.Equal(t, tt.listFlag, provider.listFlag)
			require.NotNil(t, provider.executor)
			assert.Nil(t, provider.supportedExtensions)
			assert.Nil(t, provider.initErr)
		})
	}
}

func TestGetSupportedExtensions_WithCustomCommand(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(CustomExifToolCmd, DefaultListFlag)
	mockExecutor := &MockCommandExecutor{}
	mockExecutor.On("Execute", CustomExifToolCmd, []string{DefaultListFlag}).Return([]byte(ThreeExtensionOutput), nil)

	provider := NewExtensionProvider(cfg, mockExecutor)

	extensions, err := provider.GetSupportedExtensions()

	require.NoError(t, err)
	require.Equal(t, 3, len(extensions))
	assert.True(t, extensions[".jpg"])
	assert.True(t, extensions[".png"])
	assert.True(t, extensions[".gif"])

	mockExecutor.AssertExpectations(t)
}

func TestGetSupportedExtensions_CaseInsensitive(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(DefaultExifToolCmd, DefaultListFlag)
	mockExecutor := &MockCommandExecutor{}
	mockExecutor.On("Execute", DefaultExifToolCmd, []string{DefaultListFlag}).Return([]byte(MixedCaseOutput), nil)

	provider := NewExtensionProvider(cfg, mockExecutor)

	extensions, err := provider.GetSupportedExtensions()

	require.NoError(t, err)
	assert.True(t, extensions[".jpg"])
	assert.True(t, extensions[".jpeg"])
	assert.True(t, extensions[".png"])
	assert.True(t, extensions[".gif"])

	mockExecutor.AssertExpectations(t)
}

func TestGetSupportedExtensions_MultipleCallsUseCachedResult(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(DefaultExifToolCmd, DefaultListFlag)
	mockExecutor := &MockCommandExecutor{}
	mockExecutor.On("Execute", DefaultExifToolCmd, []string{DefaultListFlag}).Return([]byte(MinimalOutput), nil).Once()

	provider := NewExtensionProvider(cfg, mockExecutor)

	for i := 0; i < 5; i++ {
		extensions, err := provider.GetSupportedExtensions()
		require.NoError(t, err)
		assert.Equal(t, 2, len(extensions))
	}

	mockExecutor.AssertExpectations(t)
}

func TestGetSupportedExtensions_WithCustomListFlag(t *testing.T) {
	t.Parallel()

	customFlag := "--list"
	cfg := createTestConfig(DefaultExifToolCmd, customFlag)
	mockExecutor := &MockCommandExecutor{}
	mockExecutor.On("Execute", DefaultExifToolCmd, []string{customFlag}).Return([]byte(MinimalOutput), nil)

	provider := NewExtensionProvider(cfg, mockExecutor)

	extensions, err := provider.GetSupportedExtensions()

	require.NoError(t, err)
	require.NotNil(t, extensions)
	assert.Equal(t, 2, len(extensions))
	mockExecutor.AssertExpectations(t)
}

func TestGetSupportedExtensions_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	cfg := createTestConfig(DefaultExifToolCmd, DefaultListFlag)
	mockExecutor := &MockCommandExecutor{}
	mockExecutor.On("Execute", DefaultExifToolCmd, []string{DefaultListFlag}).Return([]byte(MinimalOutput), nil).Once()

	provider := NewExtensionProvider(cfg, mockExecutor)

	// Run multiple goroutines concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			extensions, err := provider.GetSupportedExtensions()
			require.NoError(t, err)
			assert.Equal(t, 2, len(extensions))
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	mockExecutor.AssertExpectations(t)
}

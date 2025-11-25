package exif

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileValidator(t *testing.T) {
	t.Parallel()

	mockProvider := &MockExtensionProvider{}

	validator := NewFileValidator(mockProvider)

	require.NotNil(t, validator)
	assert.IsType(t, &FileValidator{}, validator)
}

func TestValidateFileType_SupportedFileTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filePath string
		ext      string
	}{
		{"jpeg file", "/path/to/image.jpg", ".jpg"},
		{"jpeg uppercase", "/path/to/IMAGE.JPG", ".jpg"},
		{"jpeg with dots", "/path.to/file.image.jpeg", ".jpeg"},
		{"png file", "image.png", ".png"},
		{"png uppercase", "IMAGE.PNG", ".png"},
		{"tiff file", "/tmp/scan.tiff", ".tiff"},
		{"gif file", "animated.gif", ".gif"},
		{"raw file", "photo.raw", ".raw"},
		{"cr2 file", "canon.cr2", ".cr2"},
		{"nef file", "nikon.nef", ".nef"},
		{"mp4 video", "video.mp4", ".mp4"},
		{"mov video", "movie.mov", ".mov"},
		{"pdf document", "document.pdf", ".pdf"},
		{"path with spaces", "/path with spaces/file name.jpg", ".jpg"},
		{"relative path", "./relative/path/image.png", ".png"},
		{"no path", "simple.jpg", ".jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockProvider := &MockExtensionProvider{}
			supportedExts := map[string]bool{
				".jpg":  true,
				".jpeg": true,
				".png":  true,
				".tiff": true,
				".gif":  true,
				".raw":  true,
				".cr2":  true,
				".nef":  true,
				".mp4":  true,
				".mov":  true,
				".pdf":  true,
			}
			mockProvider.On("GetSupportedExtensions").Return(supportedExts, nil)

			validator := NewFileValidator(mockProvider)

			err := validator.ValidateFileType(tt.filePath)

			require.NoError(t, err)
			mockProvider.AssertExpectations(t)
		})
	}
}

func TestValidateFileType_UnsupportedFileTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filePath string
		ext      string
	}{
		{"text file", "/path/to/document.txt", ".txt"},
		{"text uppercase", "DOCUMENT.TXT", ".txt"},
		{"executable", "program.exe", ".exe"},
		{"shell script", "script.sh", ".sh"},
		{"python script", "script.py", ".py"},
		{"go source", "main.go", ".go"},
		{"json file", "config.json", ".json"},
		{"xml file", "data.xml", ".xml"},
		{"csv file", "data.csv", ".csv"},
		{"markdown", "README.md", ".md"},
		{"yaml file", "config.yaml", ".yaml"},
		{"no extension", "filename", ""},
		{"hidden file", ".gitignore", ".gitignore"},
		{"double extension unsupported", "file.tar.gz", ".gz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockProvider := &MockExtensionProvider{}
			supportedExts := map[string]bool{
				".jpg":  true,
				".jpeg": true,
				".png":  true,
			}
			mockProvider.On("GetSupportedExtensions").Return(supportedExts, nil)

			validator := NewFileValidator(mockProvider)

			err := validator.ValidateFileType(tt.filePath)

			require.Error(t, err)

			var exifErr *Error
			require.True(t, errors.As(err, &exifErr))
			assert.Equal(t, ErrorCodeUnsupportedFileType, exifErr.Code)
			assert.Contains(t, exifErr.Message, "file type does not support EXIF data")
			assert.Contains(t, exifErr.Details, tt.ext)

			mockProvider.AssertExpectations(t)
		})
	}
}

func TestValidateFileType_ExtensionProviderError(t *testing.T) {
	t.Parallel()

	mockProvider := &MockExtensionProvider{}
	providerErr := errors.New("failed to execute exiftool")
	mockProvider.On("GetSupportedExtensions").Return(nil, providerErr)

	validator := NewFileValidator(mockProvider)

	err := validator.ValidateFileType("/path/to/image.jpg")

	require.Error(t, err)

	var exifErr *Error
	require.True(t, errors.As(err, &exifErr))
	assert.Equal(t, ErrorCodeExifToolNotFound, exifErr.Code)
	assert.Equal(t, "failed to get supported extensions", exifErr.Message)
	assert.Equal(t, providerErr.Error(), exifErr.Details)

	mockProvider.AssertExpectations(t)
}

func TestValidateFileType_EmptySupportedExtensions(t *testing.T) {
	t.Parallel()

	mockProvider := &MockExtensionProvider{}
	emptyExts := map[string]bool{}
	mockProvider.On("GetSupportedExtensions").Return(emptyExts, nil)

	validator := NewFileValidator(mockProvider)

	err := validator.ValidateFileType("/path/to/image.jpg")

	require.Error(t, err)

	var exifErr *Error
	require.True(t, errors.As(err, &exifErr))
	assert.Equal(t, ErrorCodeUnsupportedFileType, exifErr.Code)

	mockProvider.AssertExpectations(t)
}

func TestValidateFileType_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		filePath      string
		expectedExt   string
		isSupported   bool
		expectedError bool
	}{
		{
			name:          "empty filepath",
			filePath:      "",
			expectedExt:   "",
			isSupported:   false,
			expectedError: true,
		},
		{
			name:          "dot only",
			filePath:      ".",
			expectedExt:   ".",
			isSupported:   false,
			expectedError: true,
		},
		{
			name:          "multiple dots",
			filePath:      "file.backup.old.jpg",
			expectedExt:   ".jpg",
			isSupported:   true,
			expectedError: false,
		},
		{
			name:          "path with dot but no extension",
			filePath:      "./directory/filename",
			expectedExt:   "",
			isSupported:   false,
			expectedError: true,
		},
		{
			name:          "very long extension",
			filePath:      "file.verylongextension",
			expectedExt:   ".verylongextension",
			isSupported:   false,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockProvider := &MockExtensionProvider{}
			supportedExts := map[string]bool{
				".jpg":  true,
				".jpeg": true,
				".png":  true,
			}
			mockProvider.On("GetSupportedExtensions").Return(supportedExts, nil)

			validator := NewFileValidator(mockProvider)

			err := validator.ValidateFileType(tt.filePath)

			if tt.expectedError {
				require.Error(t, err)
				if tt.expectedExt != "" {
					var exifErr *Error
					require.True(t, errors.As(err, &exifErr))
					assert.Equal(t, ErrorCodeUnsupportedFileType, exifErr.Code)
				}
			} else {
				require.NoError(t, err)
			}

			mockProvider.AssertExpectations(t)
		})
	}
}

func TestValidateFileType_CaseSensitivity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		filePath   string
		shouldPass bool
	}{
		{"lowercase jpg", "image.jpg", true},
		{"uppercase JPG", "image.JPG", true},
		{"mixed case Jpg", "image.Jpg", true},
		{"mixed case jPg", "image.jPg", true},
		{"all caps JPEG", "IMAGE.JPEG", true},
		{"mixed path case", "Path/To/IMAGE.jpeg", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockProvider := &MockExtensionProvider{}
			supportedExts := map[string]bool{
				".jpg":  true,
				".jpeg": true,
			}
			mockProvider.On("GetSupportedExtensions").Return(supportedExts, nil)

			validator := NewFileValidator(mockProvider)

			err := validator.ValidateFileType(tt.filePath)

			if tt.shouldPass {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}

			mockProvider.AssertExpectations(t)
		})
	}
}

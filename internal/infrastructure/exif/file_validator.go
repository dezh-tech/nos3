package exif

import (
	"path/filepath"
	"strings"

	"nos3/internal/domain/repository/exif"
)

type FileValidator struct {
	extensionProvider exif.ExtensionProvider
}

// NewFileValidator creates a new FileValidator instance that validates file types
// against ExifTool's supported extensions using the provided ExtensionProvider
func NewFileValidator(extensionProvider exif.ExtensionProvider) exif.FileValidator {
	return &FileValidator{
		extensionProvider: extensionProvider,
	}
}

// ValidateFileType checks if the given file path has an extension supported by ExifTool.
// Returns ErrorCodeUnsupportedFileType if the file type is not supported.
// Returns ErrorCodeExifToolNotFound if unable to retrieve supported extensions.
func (f *FileValidator) ValidateFileType(filePath string) error {
	ext := strings.ToLower(filepath.Ext(filePath))

	supportedExtensions, err := f.extensionProvider.GetSupportedExtensions()
	if err != nil {
		return &ExifError{
			Code:    ErrorCodeExifToolNotFound,
			Message: "failed to get supported extensions",
			Details: err.Error(),
		}
	}

	if !supportedExtensions[ext] {
		return &ExifError{
			Code:    ErrorCodeUnsupportedFileType,
			Message: "file type does not support EXIF data",
			Details: "extension " + ext + " not supported by ExifTool",
		}
	}

	return nil
}

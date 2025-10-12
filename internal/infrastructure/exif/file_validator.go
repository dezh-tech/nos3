package exif

import (
	"nos3/internal/domain/repository/exif"
	"path/filepath"
	"strings"
)

type FileValidator struct {
	extensionProvider exif.ExtensionProvider
}

func NewFileValidator(extensionProvider exif.ExtensionProvider) exif.FileValidator {
	return &FileValidator{
		extensionProvider: extensionProvider,
	}
}

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

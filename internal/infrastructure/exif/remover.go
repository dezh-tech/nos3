package exif

import (
	"context"
	"errors"
	"time"

	"nos3/internal/domain/repository/exif"

	grpcRepository "nos3/internal/domain/repository/grpcclient"
	"nos3/pkg/logger"
)

type Remover struct {
	fileValidator exif.FileValidator
	exifProcessor exif.Processor
	timeout       time.Duration
	grpcClient    grpcRepository.IClient
}

// NewRemover creates a new Remover instance that orchestrates EXIF metadata removal.
// It uses dependency injection to receive validator and processor services.
// This is the main entry point for EXIF removal operations.
func NewRemover(
	fileValidator exif.FileValidator,
	exifProcessor exif.Processor,
	timeout time.Duration,
	grpcClient grpcRepository.IClient,
) *Remover {
	logger.Info("initializing EXIF remover")

	return &Remover{
		fileValidator: fileValidator,
		exifProcessor: exifProcessor,
		timeout:       timeout,
		grpcClient:    grpcClient,
	}
}

// RemoveExifFromFile removes EXIF metadata from the specified file.
// It first validates that the file type supports EXIF metadata.
// If the file type is unsupported, returns nil (not an error).
// For supported files, removes all EXIF data and returns any errors encountered.
func (r *Remover) RemoveExifFromFile(ctx context.Context, filePath string) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	if err := r.fileValidator.ValidateFileType(filePath); err != nil {
		var exifErr *Error
		if errors.As(err, &exifErr) && exifErr.Code == ErrorCodeUnsupportedFileType {
			return nil
		}

		return err
	}

	if err := r.exifProcessor.RemoveExifData(ctx, filePath); err != nil {
		return err
	}

	return nil
}

// Close closes the underlying EXIF processor and releases associated resources.
// Should be called when the Remover is no longer needed.
func (r *Remover) Close() error {
	return r.exifProcessor.Close()
}

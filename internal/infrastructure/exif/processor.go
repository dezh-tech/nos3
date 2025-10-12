package exif

import (
	"context"
	"nos3/internal/domain/repository/exif"
	"time"

	"github.com/barasher/go-exiftool"

	grpcRepository "nos3/internal/domain/repository/grpcclient"
	"nos3/pkg/logger"
)

type Processor struct {
	exiftool   *exiftool.Exiftool
	timeout    time.Duration
	grpcClient grpcRepository.IClient
}

// NewProcessor creates a new Processor instance that handles EXIF metadata removal.
// It initializes the ExifTool wrapper with stay-open mode for efficient processing.
// Returns an error if ExifTool initialization fails.
func NewProcessor(exiftoolCmd string,
	timeout time.Duration,
	grpcClient grpcRepository.IClient) (exif.ExifProcessor, error) {
	logger.Info("initializing exiftool for EXIF processing")

	var et *exiftool.Exiftool
	var err error

	if exiftoolCmd != "" {
		et, err = exiftool.NewExiftool(exiftool.SetExiftoolBinaryPath(exiftoolCmd))
	} else {
		et, err = exiftool.NewExiftool()
	}

	if err != nil {
		logError(context.Background(), grpcClient, "failed to initialize exiftool", err.Error())

		return nil, &ExifError{
			Code:    ErrorCodeExifToolNotFound,
			Message: "failed to initialize exiftool",
			Details: err.Error(),
		}
	}

	return &Processor{
		exiftool:   et,
		timeout:    timeout,
		grpcClient: grpcClient,
	}, nil
}

// RemoveExifData removes all EXIF metadata from the specified file.
// The operation is performed asynchronously with context cancellation support.
// Returns ErrorCodeExifRemovalFailed if metadata removal fails.
// Returns ErrorCodeTimeout if the operation exceeds the configured timeout.
func (e *Processor) RemoveExifData(ctx context.Context, filePath string) error {
	done := make(chan error, 1)

	go func() {
		defer close(done)

		fileMetadata := exiftool.FileMetadata{
			File:   filePath,
			Fields: map[string]interface{}{"All": ""},
		}

		e.exiftool.WriteMetadata([]exiftool.FileMetadata{fileMetadata})

		if fileMetadata.Err != nil {
			done <- &ExifError{
				Code:    ErrorCodeExifRemovalFailed,
				Message: "failed to remove EXIF data",
				Details: fileMetadata.Err.Error(),
			}
			return
		}

		done <- nil
	}()

	select {
	case err := <-done:
		if err != nil {
			logError(ctx, e.grpcClient, "EXIF removal failed", err.Error())
		}
		return err
	case <-ctx.Done():
		return &ExifError{
			Code:    ErrorCodeTimeout,
			Message: "EXIF removal timeout",
			Details: "operation took too long",
		}
	}
}

// Close closes the ExifTool process and releases associated resources.
// Should be called when the Processor is no longer needed.
func (e *Processor) Close() error {
	if e.exiftool != nil {
		return e.exiftool.Close()
	}
	return nil
}

// logError is a standalone function for logging errors to both local logger and gRPC client
func logError(ctx context.Context, grpcClient grpcRepository.IClient, message, details string) {
	logger.Error(message, "details", details)
	if _, logErr := grpcClient.AddLog(ctx, message, details); logErr != nil {
		logger.Error("can't send log to manager", "err", logErr)
	}
}

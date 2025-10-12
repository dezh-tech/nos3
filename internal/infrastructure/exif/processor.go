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

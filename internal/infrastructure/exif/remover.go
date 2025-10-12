package exif

import (
	"context"
	"nos3/internal/domain/repository/exif"
	"time"

	grpcRepository "nos3/internal/domain/repository/grpcclient"
	"nos3/pkg/logger"
)

type Remover struct {
	fileValidator exif.FileValidator
	exifProcessor exif.ExifProcessor
	timeout       time.Duration
	grpcClient    grpcRepository.IClient
}

func NewRemover(
	fileValidator exif.FileValidator,
	exifProcessor exif.ExifProcessor,
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

func (r *Remover) RemoveExifFromFile(ctx context.Context, filePath string) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	if err := r.fileValidator.ValidateFileType(filePath); err != nil {
		if exifErr, ok := err.(*ExifError); ok && exifErr.Code == ErrorCodeUnsupportedFileType {
			return nil
		}
		return err
	}

	if err := r.exifProcessor.RemoveExifData(ctx, filePath); err != nil {
		return err
	}

	return nil
}

func (r *Remover) Close() error {
	return r.exifProcessor.Close()
}

package exif

import "context"

// ExifProcessor handles EXIF removal operations
type ExifProcessor interface {
	RemoveExifData(ctx context.Context, filePath string) error
	Close() error
}

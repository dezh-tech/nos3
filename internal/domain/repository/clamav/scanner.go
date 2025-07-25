package clamav

import (
	"context"
	"io"
	"nos3/internal/domain/entity"
)

// Scanner defines the interface for scanning files for malware.
type Scanner interface {
	ScanStream(ctx context.Context, reader io.Reader) (entity.MalwareScanResult, error)
}

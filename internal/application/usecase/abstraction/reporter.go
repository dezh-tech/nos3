package abstraction

import "context"

// Reporter defines the interface for reporting blobs.
type Reporter interface {
	ReportBlob(ctx context.Context, pubKey string, blobHashes []string,
		reportType, eventID, content, serverURL string) error
}

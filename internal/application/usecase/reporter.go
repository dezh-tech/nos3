package usecase

import (
	"context"
	"fmt"

	grpcRepository "nos3/internal/domain/repository/grpcclient"
	"nos3/pkg/logger"
)

// BlobReporter implements the Reporter abstraction for sending blob reports.
type BlobReporter struct {
	grpcClient grpcRepository.IClient
}

// NewBlobReporter creates a new BlobReporter usecase.
func NewBlobReporter(grpcClient grpcRepository.IClient) *BlobReporter {
	return &BlobReporter{
		grpcClient: grpcClient,
	}
}

// ReportBlob sends a blob report to the manager service via gRPC.
func (r *BlobReporter) ReportBlob(ctx context.Context, pubKey string, blobHashes []string,
	reportType, eventID, content, serverURL string,
) error {
	resp, err := r.grpcClient.AddReport(ctx, pubKey, blobHashes, reportType, eventID, content, serverURL)
	if err != nil {
		logger.Error("failed to send blob report via gRPC", "err", err, "pubKey", pubKey, "blobHashes", blobHashes)

		return fmt.Errorf("failed to send blob report: %w", err)
	}

	if !resp.Success {
		errMsg := "unknown error"
		if resp.Message != nil {
			errMsg = *resp.Message
		}
		logger.Error("blob report not successful", "message", errMsg, "pubKey", pubKey, "blobHashes", blobHashes)

		return fmt.Errorf("blob report failed: %s", errMsg)
	}

	logger.Info("blob report sent successfully", "pubKey", pubKey, "blobHashes", blobHashes)

	return nil
}

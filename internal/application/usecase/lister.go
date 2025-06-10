package usecase

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"nos3/internal/domain/dto"
	"nos3/internal/domain/repository/database"
)

// Lister implements the Lister abstraction for retrieving blob information.
type Lister struct {
	lister         database.Lister
	defaultAddress string
}

// NewLister creates a new Lister usecase.
func NewLister(lister database.Lister, address string) *Lister {
	return &Lister{
		lister:         lister,
		defaultAddress: address,
	}
}

// ListBlobs retrieves blobs by author and optional time filters from the database.
func (l *Lister) ListBlobs(ctx context.Context, pubKey string, since,
	until *time.Time) ([]dto.BlobDescriptor, int, error) {
	blobs, err := l.lister.GetByAuthor(ctx, pubKey, since, until)
	if err != nil {
		return nil, http.StatusNotFound, errors.New("failed to retrieve blobs")
	}

	descriptors := make([]dto.BlobDescriptor, 0, len(blobs))
	for i := range blobs {
		descriptors = append(descriptors, dto.BlobDescriptor{
			URL:      fmt.Sprintf("%s/%s", l.defaultAddress, blobs[i].ID),
			Sha256:   blobs[i].ID,
			Size:     blobs[i].Size,
			FileType: blobs[i].BlobType,
			Uploaded: blobs[i].UploadTime.Unix(),
		})
	}

	return descriptors, http.StatusOK, nil
}

package abstraction

import "context"

// Deleter defines the interface for deleting blob information.
type Deleter interface {
	DeleteBlob(ctx context.Context, sha256 string) (int, error)
}

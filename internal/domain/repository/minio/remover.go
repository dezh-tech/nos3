package minio

import "context"

type Remover interface {
	Remove(ctx context.Context, bucketName, objectName string) error
}

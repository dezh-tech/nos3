package database

import "context"

type Remover interface {
	RemoveByHash(ctx context.Context, hash string) error
}

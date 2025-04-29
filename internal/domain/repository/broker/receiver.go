package broker

import "context"

type Receiver interface {
	Messages(ctx context.Context) (<-chan Message, error)
}

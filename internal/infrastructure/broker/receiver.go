package broker

import (
	"context"
	"errors"
	"time"

	"nos3/internal/domain/repository/broker"

	"github.com/redis/go-redis/v9"
)

type Receiver struct {
	redis     *redis.Client
	stream    string
	group     string
	blockTime time.Duration
}

func NewReceiver(client *Client) *Receiver {
	return &Receiver{
		redis:     client.redis,
		stream:    client.stream,
		group:     client.group,
		blockTime: 5 * time.Second,
	}
}

func (r *Receiver) Messages(ctx context.Context, consumerName string) (<-chan broker.Message, error) {
	if r.redis == nil {
		return nil, errors.New("redis not initialized")
	}

	out := make(chan broker.Message)

	go func() {
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				entries, err := r.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
					Group:    r.group,
					Consumer: consumerName,
					Streams:  []string{r.stream, ">"},
					Count:    1,
					Block:    r.blockTime,
				}).Result()

				if err != nil && !errors.Is(err, redis.Nil) {
					continue
				}

				for _, stream := range entries {
					for _, msg := range stream.Messages {
						body := msg.Values["body"].(string)
						out <- &RedisMessage{
							stream:      r.stream,
							group:       r.group,
							consumer:    consumerName,
							id:          msg.ID,
							body:        body,
							redisClient: r.redis,
						}
					}
				}
			}
		}
	}()

	return out, nil
}

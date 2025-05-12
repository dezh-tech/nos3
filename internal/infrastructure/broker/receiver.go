package broker

import (
	"context"
	"errors"
	"time"

	"github.com/dezh-tech/immortal/pkg/logger"
	"github.com/redis/go-redis/v9"

	"nos3/internal/domain/repository/broker"
	grpcRepository "nos3/internal/domain/repository/grpcclient"
)

type Receiver struct {
	redis      *redis.Client
	stream     string
	group      string
	blockTime  time.Duration
	grpcClient grpcRepository.IClient
}

func NewReceiver(client *Client, grpcClient grpcRepository.IClient) *Receiver {
	return &Receiver{
		redis:      client.redis,
		stream:     client.stream,
		group:      client.group,
		blockTime:  5 * time.Second,
		grpcClient: grpcClient,
	}
}
func (r *Receiver) Messages(ctx context.Context, consumerName string) (<-chan broker.Message, error) {
	if r.redis == nil {
		logger.Error("redis client is nil in receiver")

		return nil, errors.New("redis not initialized")
	}

	out := make(chan broker.Message)
	go r.consumeLoop(ctx, out, consumerName)

	return out, nil
}

func (r *Receiver) consumeLoop(ctx context.Context, out chan broker.Message, consumerName string) {
	defer close(out)

	for {
		select {
		case <-ctx.Done():
			logger.Error("message receiving context cancelled")

			return
		default:
			r.readAndEmit(ctx, out, consumerName)
		}
	}
}

func (r *Receiver) readAndEmit(ctx context.Context, out chan broker.Message, consumerName string) {
	entries, err := r.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    r.group,
		Consumer: consumerName,
		Streams:  []string{r.stream, ">"},
		Count:    1,
		Block:    r.blockTime,
	}).Result()

	if err != nil && !errors.Is(err, redis.Nil) {
		if _, logErr := r.grpcClient.AddLog(ctx, "failed to read from redis stream group", err.Error()); logErr != nil {
			logger.Error("can't send log to manager", "err", logErr)
		}

		return
	}

	for _, stream := range entries {
		for _, msg := range stream.Messages {
			body, ok := msg.Values["body"].(string)
			if !ok {
				logger.Error("invalid body type in redis message", "id", msg.ID)

				continue
			}
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

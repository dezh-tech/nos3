package broker

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type Publisher struct {
	redis  *redis.Client
	stream string
	timout time.Duration
}

func NewPublisher(client *Client, cfg PublisherConfig) *Publisher {
	return &Publisher{
		redis:  client.redis,
		stream: client.stream,
		timout: time.Duration(cfg.Timeout) * time.Millisecond,
	}
}

func (p *Publisher) Publish(ctx context.Context, message string) error {
	if p.redis == nil {
		return errors.New("redis not initialized")
	}

	ctx, cancel := context.WithTimeout(ctx, p.timout)
	defer cancel()

	return p.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: p.stream,
		Values: map[string]interface{}{"body": message},
	}).Err()
}

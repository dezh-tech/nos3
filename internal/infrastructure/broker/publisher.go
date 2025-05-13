package broker

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"

	grpcRepository "nos3/internal/domain/repository/grpcclient"
	"nos3/pkg/logger"
)

type Publisher struct {
	redis      *redis.Client
	stream     string
	timout     time.Duration
	grpcClient grpcRepository.IClient
}

func NewPublisher(client *Client, cfg PublisherConfig, grpcClient grpcRepository.IClient) *Publisher {
	return &Publisher{
		redis:      client.redis,
		stream:     client.stream,
		timout:     time.Duration(cfg.Timeout) * time.Millisecond,
		grpcClient: grpcClient,
	}
}

func (p *Publisher) Publish(ctx context.Context, message string) error {
	if p.redis == nil {
		logger.Error("redis client is nil during publish")

		return errors.New("redis not initialized")
	}

	ctx, cancel := context.WithTimeout(ctx, p.timout)
	defer cancel()

	err := p.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: p.stream,
		Values: map[string]interface{}{"body": message},
	}).Err()
	if err != nil {
		if _, logErr := p.grpcClient.AddLog(ctx, "failed to publish message to redis stream", err.Error()); logErr != nil {
			logger.Error("can't send log to manager", "err", logErr)
		}
	}

	return err
}

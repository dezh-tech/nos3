package broker

import (
	"context"

	grpcRepository "nos3/internal/domain/repository/grpcclient"

	"github.com/dezh-tech/immortal/pkg/logger"
	"github.com/redis/go-redis/v9"
)

type RedisMessage struct {
	stream      string
	group       string
	consumer    string
	id          string
	body        string
	redisClient *redis.Client
	grpcClient  grpcRepository.IClient
}

func (m *RedisMessage) Body() string {
	return m.body
}

func (m *RedisMessage) Ack() error {
	err := m.redisClient.XAck(context.Background(), m.stream, m.group, m.id).Err()
	if err != nil {
		if _, logErr := m.grpcClient.AddLog(context.Background(), "failed to ack message", err.Error()); logErr != nil {
			logger.Error("can't send log to manager", "err", logErr)
		}
	}

	return err
}

func (m *RedisMessage) Nack() error {
	return nil
}

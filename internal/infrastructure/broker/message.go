package broker

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisMessage struct {
	stream      string
	group       string
	consumer    string
	id          string
	body        string
	redisClient *redis.Client
}

func (m *RedisMessage) Body() string {
	return m.body
}

func (m *RedisMessage) Ack() error {
	return m.redisClient.XAck(context.Background(), m.stream, m.group, m.id).Err()
}

func (m *RedisMessage) Nack() error {
	return nil
}

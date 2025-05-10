package broker

import (
	"context"
	"fmt"
	"net"
	"nos3/internal/infrastructure/grpcclient"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	RedisImage = "redis:7-alpine"
	StreamName = "test-stream"
	GroupName  = "test-group"
	Consumer   = "test-consumer"
)

func setupRedis(t *testing.T) (string, func()) {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        RedisImage,
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp"),
	}

	redisC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("failed to start Redis container: %v", err)
	}

	host, err := redisC.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get Redis container host: %v", err)
	}

	port, err := redisC.MappedPort(ctx, "6379")
	if err != nil {
		t.Fatalf("failed to get Redis container port: %v", err)
	}

	hostPort := net.JoinHostPort(host, port.Port())
	uri := fmt.Sprintf("redis://%s", hostPort)

	return uri, func() {
		_ = redisC.Terminate(ctx)
	}
}

func TestPublish(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		messages  []string
		expectLen int
	}{
		{"one message", []string{"hello"}, 1},
		{"empty message", []string{""}, 1},
		{"ten messages", []string{"msg1", "msg2", "msg3", "msg4", "msg5", "msg6", "msg7", "msg8", "msg9", "msg10"}, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uri, terminate := setupRedis(t)
			defer terminate()

			client, err := NewClient(Config{
				URI:        uri,
				StreamName: StreamName,
				GroupName:  GroupName,
			}, &grpcclient.Client{})
			if err != nil {
				t.Fatalf("failed to create Redis client: %v", err)
			}
			defer client.Close()

			publisher := NewPublisher(client, PublisherConfig{Timeout: 1000})

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			for _, msg := range tt.messages {
				err := publisher.Publish(ctx, msg)
				assert.NoError(t, err)
			}

			read, err := client.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    GroupName,
				Consumer: Consumer,
				Streams:  []string{StreamName, ">"},
				Count:    int64(tt.expectLen),
				Block:    2 * time.Second,
			}).Result()
			assert.NoError(t, err)
			assert.Len(t, read, 1)
			assert.Len(t, read[0].Messages, tt.expectLen)

			for i, msg := range tt.messages {
				assert.Equal(t, msg, read[0].Messages[i].Values["body"])
			}
		})
	}
}

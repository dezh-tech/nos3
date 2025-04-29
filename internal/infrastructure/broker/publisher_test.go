package broker

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	RabbitUser  = "guest"
	RabbitPass  = "guest"
	RabbitQueue = "test-queue"
)

func assertMessageInQueue(t *testing.T, client *Client, expected string) {
	t.Helper()
	deliveries, err := client.channel.Consume(
		client.queueName,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		t.Fatalf("consume failed: %v", err)
	}

	select {
	case d := <-deliveries:
		if string(d.Body) != expected {
			t.Errorf("Expected %q, got %q", expected, d.Body)
		}
	case <-time.After(5 * time.Second):
		t.Errorf("Timed out waiting for message")
	}
}

func setupRabbitMQ(t *testing.T) string {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "rabbitmq:3.7.25-management-alpine",
		ExposedPorts: []string{"5672/tcp"},
		WaitingFor:   wait.ForListeningPort("5672/tcp").WithStartupTimeout(30 * time.Second),
		Env: map[string]string{
			"RABBITMQ_DEFAULT_USER": RabbitUser,
			"RABBITMQ_DEFAULT_PASS": RabbitPass,
		},
	}

	rmqContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatal("Failed to start RabbitMQ container:", err)
	}

	host, err := rmqContainer.Host(ctx)
	if err != nil {
		t.Fatal("Failed to get container host:", err)
	}

	port, err := rmqContainer.MappedPort(ctx, "5672")
	if err != nil {
		t.Fatal("Failed to get mapped port:", err)
	}

	hostPort := net.JoinHostPort(host, port.Port())
	uri := fmt.Sprintf("amqp://%s:%s@%s", RabbitUser, RabbitPass, hostPort)

	return uri
}

func TestPublish(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		msg  string
	}{
		{"valid publish", "hello"},
		{"empty message", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			uri := setupRabbitMQ(t)

			client, err := NewClient(Config{
				URI:       uri,
				QueueName: RabbitQueue,
			})

			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()

			publisher := NewPublisher(client, PublisherConfig{Timeout: 2000})

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err = publisher.Publish(ctx, tt.msg)
			assert.NoError(t, err)
			assertMessageInQueue(t, client, tt.msg)
		})
	}
}

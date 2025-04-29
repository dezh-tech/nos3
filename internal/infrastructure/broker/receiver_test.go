package broker

import (
	"context"
	"errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func publishForTest(t *testing.T, ctx context.Context, message string, client *Client, timeout time.Duration) error {
	t.Helper()
	if client.channel == nil {
		return errors.New("channel is not initialized")
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return client.channel.PublishWithContext(
		ctx,
		"",
		client.queueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  "text/plain",
			Body:         []byte(message),
			Timestamp:    time.Now(),
			DeliveryMode: amqp.Persistent,
		},
	)
}

func TestMessages(t *testing.T) {
	t.Helper()

	uri := setupRabbitMQ(t)

	client, err := NewClient(Config{
		URI:       uri,
		QueueName: RabbitQueue,
	})
	defer client.Close()
	if err != nil {
		t.Fatal(err)
	}

	receiver := NewReceiver(client)

	msgs := []string{"one", "two", "three"}

	for _, msg := range msgs {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := publishForTest(t, ctx, msg, client, time.Duration(5000)*time.Millisecond)
		cancel()
		assert.NoError(t, err)
	}

	ctx := context.Background()
	out, err := receiver.Messages(ctx)
	assert.NoError(t, err)

	received := make([]string, 0, len(msgs))
	timeout := time.After(3 * time.Second)

Loop:
	for {
		select {
		case m, ok := <-out:
			if !ok {
				break Loop
			}
			received = append(received, m.Body())
			_ = m.Ack()
			if len(received) == len(msgs) {
				break Loop
			}
		case <-timeout:
			t.Fatal("Timed out waiting for messages")
		}
	}

	assert.ElementsMatch(t, msgs, received)
}

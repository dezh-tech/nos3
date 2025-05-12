package broker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/stretchr/testify/assert"
)

func publishMessages(t *testing.T, client *Client, messages []string) error {
	t.Helper()

	if client.redis == nil {
		return errors.New("redis not initialized")
	}

	for _, msg := range messages {
		err := client.redis.XAdd(context.Background(), &redis.XAddArgs{
			Stream: client.stream,
			Values: map[string]interface{}{"body": msg},
		}).Err()
		if err != nil {
			return err
		}
	}

	return nil
}

func TestMessages_SingleAndMultipleMessages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		payloads []string
	}{
		{"single message", []string{"hello"}},
		{"empty message", []string{""}},
		{"multiple messages", []string{"a", "b", "c", "d", "e"}},
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
			}, &MockGRPC{})
			assert.NoError(t, err)
			defer client.Close()

			err = publishMessages(t, client, tt.payloads)
			assert.NoError(t, err)

			receiver := NewReceiver(client, &MockGRPC{})
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			ch, err := receiver.Messages(ctx, Consumer)
			assert.NoError(t, err)

			received := make([]string, 0, len(tt.payloads))
			for range tt.payloads {
				msg := <-ch
				received = append(received, msg.Body())
				assert.NoError(t, msg.Ack())
			}

			assert.ElementsMatch(t, tt.payloads, received)
		})
	}
}

func TestMessages_ConcurrentConsumers(t *testing.T) {
	t.Parallel()
	uri, terminate := setupRedis(t)
	defer terminate()

	client, err := NewClient(Config{
		URI:        uri,
		StreamName: StreamName,
		GroupName:  GroupName,
	}, &MockGRPC{})
	assert.NoError(t, err)
	defer client.Close()

	totalMessages := 100
	workers := 5
	messages := make([]string, totalMessages)
	for i := 0; i < totalMessages; i++ {
		messages[i] = fmt.Sprintf("msg-%d", i)
	}

	err = publishMessages(t, client, messages)
	assert.NoError(t, err)

	received := make(chan string, totalMessages)
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	receiver := NewReceiver(client, &MockGRPC{})

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ch, err := receiver.Messages(ctx, fmt.Sprintf("consumer-%d", id))
			if err != nil {
				return
			}
			for msg := range ch {
				received <- msg.Body()
				_ = msg.Ack()
			}
		}(i)
	}

	wg.Wait()
	close(received)

	seen := make(map[string]bool)
	for msg := range received {
		assert.False(t, seen[msg], "duplicate message received: %s", msg)
		seen[msg] = true
	}
	assert.Len(t, seen, totalMessages)
}

func TestMessages_ContextCancel(t *testing.T) {
	t.Parallel()
	uri, terminate := setupRedis(t)
	defer terminate()

	client, err := NewClient(Config{
		URI:        uri,
		StreamName: StreamName,
		GroupName:  GroupName,
	}, &MockGRPC{})
	assert.NoError(t, err)
	defer client.Close()

	receiver := NewReceiver(client, &MockGRPC{})
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	ch, err := receiver.Messages(ctx, "consumer-cancel")
	assert.NoError(t, err)
	_, ok := <-ch
	assert.False(t, ok, "expected channel to be closed due to context cancel")
}

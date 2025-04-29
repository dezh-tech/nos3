package broker

import (
	"context"
	"errors"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	rabbitMQ *Client
	timeout  time.Duration
}

func NewPublisher(rabbitMQ *Client, cfg PublisherConfig) *Publisher {
	return &Publisher{
		rabbitMQ: rabbitMQ,
		timeout:  time.Duration(cfg.Timeout) * time.Millisecond,
	}
}

func (p *Publisher) Publish(ctx context.Context, message string) error {
	if p.rabbitMQ.channel == nil {
		return errors.New("channel is not initialized")
	}

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	return p.rabbitMQ.channel.PublishWithContext(
		ctx,
		"",
		p.rabbitMQ.queueName,
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

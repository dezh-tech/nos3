package broker

import amqp "github.com/rabbitmq/amqp091-go"

type Client struct {
	conn      *amqp.Connection
	channel   *amqp.Channel
	queueName string
}

// NewClient creates and initializes a RabbitMQ client with a queue.
// The caller is responsible for calling `defer client.Close()` to release resources properly.
func NewClient(cfg Config) (*Client, error) {
	conn, err := amqp.Dial(cfg.URI)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()

		return nil, err
	}

	_, err = ch.QueueDeclare(
		cfg.QueueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ch.Close()
		conn.Close()

		return nil, err
	}

	_ = ch.Qos(
		1,
		0,
		false,
	)

	return &Client{
		conn:      conn,
		channel:   ch,
		queueName: cfg.QueueName,
	}, nil
}

func (c *Client) Close() {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

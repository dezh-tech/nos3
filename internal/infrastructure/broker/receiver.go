package broker

import (
	"context"
	"errors"
	"nos3/internal/domain/repository/broker"
)

type Receiver struct {
	rabbitMQ *Client
}

func NewReceiver(rabbitMQ *Client) *Receiver {
	return &Receiver{
		rabbitMQ: rabbitMQ,
	}
}

// Messages starts a long-running task that continuously reads messages.
// Be careful: if the provided context has a short timeout or is canceled early,
// message consumption will stop and resources must be cleaned up properly.
// NOTE: AutoAck is disabled. Caller must manually call Ack/Nack on each message after processing.
func (r *Receiver) Messages(ctx context.Context) (<-chan broker.Message, error) {

	if r.rabbitMQ.channel == nil {
		return nil, errors.New("channel is not initialized")
	}

	deliveries, err := r.rabbitMQ.channel.Consume(
		r.rabbitMQ.queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)

	if err != nil {
		return nil, err
	}

	out := make(chan broker.Message)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-deliveries:
				if !ok {
					return
				}
				out <- &RabbitMessage{
					body:     string(msg.Body),
					delivery: msg,
				}
			}
		}
	}()
	return out, nil
}

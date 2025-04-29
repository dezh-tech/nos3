package broker

import amqp "github.com/rabbitmq/amqp091-go"

type RabbitMessage struct {
	body     string
	delivery amqp.Delivery
}

func (m *RabbitMessage) Body() string {
	return m.body
}

func (m *RabbitMessage) Ack() error {
	return m.delivery.Ack(false)
}

func (m *RabbitMessage) Nack() error {
	return m.delivery.Nack(false, true)
}

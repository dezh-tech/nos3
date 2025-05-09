package broker

type Message interface {
	Body() string
	Ack() error
	Nack() error
}

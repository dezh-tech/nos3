package broker

type Config struct {
	URI       string
	QueueName string `yaml:"queue_name"`
}

type PublisherConfig struct {
	Timeout int `yaml:"timeout_in_ms"`
}

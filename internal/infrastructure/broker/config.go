package broker

type Config struct {
	URI        string
	StreamName string `yaml:"stream_name"`
	GroupName  string `yaml:"group_name"`
}

type PublisherConfig struct {
	Timeout int `yaml:"timeout_in_ms"`
}

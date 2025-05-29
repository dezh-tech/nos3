package grpcclient

type ClientConfig struct {
	Endpoint  string `yaml:"endpoint"`
	Region    string `yaml:"region"`
	Heartbeat uint32 `yaml:"heartbeat_in_second"`
}

type ServerConfig struct {
	Bind string `yaml:"bind"`
	Port uint16 `yaml:"port"`
}

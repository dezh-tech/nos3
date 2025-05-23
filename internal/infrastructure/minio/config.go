package minio

type ClientConfig struct {
	AccessKey string
	SecretKey string
	Endpoint  string `yaml:"endpoint"`
}

type UploaderConfig struct {
	Timeout int64  `yaml:"timeout_in_ms"`
	Bucket  string `yaml:"bucket"`
}

type RemoverConfig struct {
	Timeout int64 `yaml:"timeout_in_ms"`
}

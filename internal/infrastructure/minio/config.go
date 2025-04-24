package minio

type Config struct {
	Timeout int64  `yaml:"timeout_in_ms"`
	Bucket  string `yaml:"bucket"`
}

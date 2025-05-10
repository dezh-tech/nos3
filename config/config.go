package config

import (
	"os"

	"nos3/internal/infrastructure/grpcclient"

	"nos3/internal/infrastructure/broker"

	"nos3/internal/infrastructure/database"

	"github.com/dezh-tech/immortal/pkg/logger"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"

	"nos3/internal/infrastructure/minio"
)

// Config represents the configs used by services on system.
type Config struct {
	Environment     string                 `yaml:"environment"`
	MinIOClient     minio.ClientConfig     `yaml:"minio_client"`
	MinIOUploader   minio.UploaderConfig   `yaml:"minio_uploader"`
	DBConfig        database.Config        `yaml:"db_config"`
	BrokerConfig    broker.Config          `yaml:"redis_broker_config"`
	PublisherConfig broker.PublisherConfig `yaml:"publisher_config"`
	GRPCClient      grpcclient.Config      `yaml:"manager"`
	Logger          logger.Config          `yaml:"logger"`
}

func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, Error{
			reason: err.Error(),
		}
	}
	defer file.Close()

	config := &Config{}

	decoder := yaml.NewDecoder(file)

	if err := decoder.Decode(config); err != nil {
		return nil, Error{
			reason: err.Error(),
		}
	}

	if config.Environment != "prod" {
		if err := godotenv.Load(); err != nil {
			return nil, Error{
				reason: err.Error(),
			}
		}
	}

	config.MinIOClient.AccessKey = os.Getenv("MINIO_ROOT_USER")
	config.MinIOClient.SecretKey = os.Getenv("MINIO_ROOT_PASSWORD")
	config.DBConfig.URI = os.Getenv("DATABASE_URI")
	config.BrokerConfig.URI = os.Getenv("BROKER_URI")

	if err = config.basicCheck(); err != nil {
		return nil, Error{
			reason: err.Error(),
		}
	}

	return config, nil
}

// basicCheck validates the basic stuff in config.
func (c *Config) basicCheck() error {
	return nil
}

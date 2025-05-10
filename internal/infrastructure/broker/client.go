package broker

import (
	"context"

	"nos3/internal/infrastructure/grpcclient"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	redis      *redis.Client
	stream     string
	group      string
	grpcClient *grpcclient.Client
}

func NewClient(cfg Config, grpcClient *grpcclient.Client) (*Client, error) {
	opt, err := redis.ParseURL(cfg.URI)
	if err != nil {
		return nil, err
	}

	rdb := redis.NewClient(opt)
	ctx := context.Background()

	err = rdb.XGroupCreateMkStream(ctx, cfg.StreamName, cfg.GroupName, "$").Err()
	if err != nil && !isBusyGroup(err) {
		return nil, err
	}

	return &Client{
		redis:      rdb,
		stream:     cfg.StreamName,
		group:      cfg.GroupName,
		grpcClient: grpcClient,
	}, nil
}

func (c *Client) Close() error {
	return c.redis.Close()
}

func isBusyGroup(err error) bool {
	return err != nil && err.Error() == "BUSYGROUP Consumer Group name already exists"
}

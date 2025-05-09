package broker

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	redis  *redis.Client
	stream string
	group  string
}

func NewClient(cfg Config) (*Client, error) {
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
		redis:  rdb,
		stream: cfg.StreamName,
		group:  cfg.GroupName,
	}, nil
}

func (c *Client) Close() error {
	return c.redis.Close()
}

func isBusyGroup(err error) bool {
	return err != nil && err.Error() == "BUSYGROUP Consumer Group name already exists"
}

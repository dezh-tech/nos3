package broker

import (
	"context"

	"github.com/dezh-tech/immortal/pkg/logger"
	"github.com/redis/go-redis/v9"

	"nos3/internal/domain/repository/grpcclient"
)

type Client struct {
	redis      *redis.Client
	stream     string
	group      string
	grpcClient grpcclient.IClient
}

func NewClient(cfg Config, grpcClient grpcclient.IClient) (*Client, error) {
	logger.Info("connecting to redis broker")

	opt, err := redis.ParseURL(cfg.URI)
	if err != nil {
		if _, logErr := grpcClient.AddLog(context.Background(), "failed to parse redis URI", err.Error()); logErr != nil {
			logger.Error("can't send log to manager", "err", logErr)
		}

		return nil, err
	}

	rdb := redis.NewClient(opt)
	ctx := context.Background()

	err = rdb.XGroupCreateMkStream(ctx, cfg.StreamName, cfg.GroupName, "$").Err()
	if err != nil && !isBusyGroup(err) {
		if _, logErr := grpcClient.AddLog(ctx, "failed to create redis stream group", err.Error()); logErr != nil {
			logger.Error("can't send log to manager", "err", logErr)
		}

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
	err := c.redis.Close()
	if err != nil {
		if _, logErr := c.grpcClient.AddLog(context.Background(), "failed to close redis client",
			err.Error()); logErr != nil {
			logger.Error("can't send log to manager", "err", logErr)
		}
	}

	return err
}

func isBusyGroup(err error) bool {
	return err != nil && err.Error() == "BUSYGROUP Consumer Group name already exists"
}

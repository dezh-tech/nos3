package grpcclient

import (
	"context"

	mpb "nos3/internal/infrastructure/grpcclient/gen"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	RegistryService mpb.ServiceRegistryClient
	LogService      mpb.LogClient
	config          Config
	conn            *grpc.ClientConn
}

func New(endpoint string, cfg Config) (*Client, error) {
	conn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &Client{
		RegistryService: mpb.NewServiceRegistryClient(conn),
		LogService:      mpb.NewLogClient(conn),
		config:          cfg,
		conn:            conn,
	}, nil
}

func (c *Client) RegisterService(ctx context.Context,
	port, region string,
) (*mpb.RegisterServiceResponse, error) {
	return c.RegistryService.RegisterService(ctx, &mpb.RegisterServiceRequest{
		Type:                   mpb.ServiceTypeEnum_STORAGE,
		Port:                   port,
		HeartbeatDurationInSec: c.config.Heartbeat,
		Region:                 region,
	})
}

func (c *Client) AddLog(ctx context.Context, msg, stack string) (*mpb.AddLogResponse, error) {
	return c.LogService.AddLog(ctx, &mpb.AddLogRequest{
		Message: msg,
		Stack:   stack,
	})
}

package grpcclient

import (
	"context"

	mpb "nos3/internal/infrastructure/grpcclient/gen"
)

type IClient interface {
	RegisterService(ctx context.Context,
		port, region string,
	) (*mpb.RegisterServiceResponse, error)
	AddLog(ctx context.Context, msg, stack string) (*mpb.AddLogResponse, error)
}

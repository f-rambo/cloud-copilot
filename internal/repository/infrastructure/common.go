package infrastructure

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/f-rambo/cloud-copilot/utils"
)

func getInfrastructureClusterServiceConfig(conf *conf.Bootstrap) *conf.Service {
	for _, service := range conf.Services {
		if service.Name == ServiceNameInfrastructure {
			return service
		}
	}
	return nil
}

func coonGrpc(ctx context.Context, conf *conf.Bootstrap) (*utils.GrpcConn, error) {
	service := getInfrastructureClusterServiceConfig(conf)
	return new(utils.GrpcConn).OpenGrpcConn(ctx, service.Addr, service.Port, service.Timeout)
}

package clusterruntime

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/f-rambo/cloud-copilot/utils"
)

func getClusterRuntimeProjectServiceConfig(conf *conf.Bootstrap) *conf.Service {
	for _, service := range conf.Services {
		if service.Name == ServiceNameClusterRuntime {
			return service
		}
	}
	return nil
}

func connGrpc(ctx context.Context, conf *conf.Bootstrap) (*utils.GrpcConn, error) {
	service := getClusterRuntimeProjectServiceConfig(conf)
	return new(utils.GrpcConn).OpenGrpcConn(ctx, service.Addr, service.Port, service.Timeout)
}

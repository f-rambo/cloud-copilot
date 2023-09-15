package server

import (
	appv1 "github.com/f-rambo/ocean/api/app/v1"
	clusterv1 "github.com/f-rambo/ocean/api/cluster/v1"
	servicev1 "github.com/f-rambo/ocean/api/service/v1"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server, cluster *service.ClusterService, app *service.AppService, services *service.ServicesService, logger log.Logger) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
		),
	}
	netWork := c.GRPC.GetNetwork()
	if netWork != "" {
		opts = append(opts, grpc.Network(netWork))
	}
	addr := c.GRPC.GetAddr()
	if addr != "" {
		opts = append(opts, grpc.Address(addr))
	}
	srv := grpc.NewServer(opts...)
	clusterv1.RegisterClusterServiceServer(srv, cluster)
	appv1.RegisterAppServiceServer(srv, app)
	servicev1.RegisterServiceServiceServer(srv, services)
	return srv
}

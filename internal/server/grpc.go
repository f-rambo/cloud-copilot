package server

import (
	appv1 "ocean/api/app/v1"
	clusterv1 "ocean/api/cluster/v1"
	helloworldv1 "ocean/api/helloworld/v1"
	"ocean/internal/conf"
	"ocean/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server, greeter *service.GreeterService, cluster *service.ClusterService, app *service.AppService, logger log.Logger) *grpc.Server {
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
	helloworldv1.RegisterGreeterServer(srv, greeter)
	clusterv1.RegisterClusterServiceServer(srv, cluster)
	appv1.RegisterAppServiceServer(srv, app)
	return srv
}

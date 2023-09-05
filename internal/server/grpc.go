package server

import (
	appv1 "ocean/api/app/v1"
	clusterv1 "ocean/api/cluster/v1"
	helloworldv1 "ocean/api/helloworld/v1"
	infrav1 "ocean/api/infra/v1"
	"ocean/internal/conf"
	"ocean/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server, greeter *service.GreeterService, infra *service.InfraService,
	cluster *service.ClusterService, app *service.AppService, logger log.Logger) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
		),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	helloworldv1.RegisterGreeterServer(srv, greeter)
	infrav1.RegisterInfraServer(srv, infra)
	clusterv1.RegisterClusterServer(srv, cluster)
	appv1.RegisterAppServiceServer(srv, app)
	return srv
}

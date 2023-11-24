package server

import (
	appv1alpha1 "github.com/f-rambo/ocean/api/app/v1alpha1"
	clusterv1alpha1 "github.com/f-rambo/ocean/api/cluster/v1alpha1"
	servicev1alpha1 "github.com/f-rambo/ocean/api/service/v1alpha1"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server,
	cluster *service.ClusterService,
	app *service.AppService,
	services *service.ServicesService,
	user *service.UserService,
	logger log.Logger) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			NewAuthServer(user),
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
	clusterv1alpha1.RegisterClusterServiceServer(srv, cluster)
	appv1alpha1.RegisterAppServiceServer(srv, app)
	servicev1alpha1.RegisterServiceServiceServer(srv, services)
	return srv
}

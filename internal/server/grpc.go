package server

import (
	appv1alpha1 "github.com/f-rambo/ocean/api/app/v1alpha1"
	clusterv1alpha1 "github.com/f-rambo/ocean/api/cluster/v1alpha1"
	projectv1alpha1 "github.com/f-rambo/ocean/api/project/v1alpha1"
	servicev1alpha1 "github.com/f-rambo/ocean/api/service/v1alpha1"
	userv1alpha1 "github.com/f-rambo/ocean/api/user/v1alpha1"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/internal/interfaces"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server,
	cluster *interfaces.ClusterInterface,
	app *interfaces.AppInterface,
	services *interfaces.ServicesInterface,
	user *interfaces.UserInterface,
	project *interfaces.ProjectInterface,
	logger log.Logger) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			selector.Server(NewAuthServer(user)).Match(NewWhiteListMatcher()).Build(),
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
	clusterv1alpha1.RegisterClusterInterfaceServer(srv, cluster)
	appv1alpha1.RegisterAppInterfaceServer(srv, app)
	servicev1alpha1.RegisterServiceInterfaceServer(srv, services)
	userv1alpha1.RegisterUserInterfaceServer(srv, user)
	projectv1alpha1.RegisterProjectServiceServer(srv, project)
	return srv
}

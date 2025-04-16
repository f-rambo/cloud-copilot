package server

import (
	"time"

	appv1alpha1 "github.com/f-rambo/cloud-copilot/api/app/v1alpha1"
	clusterv1alpha1 "github.com/f-rambo/cloud-copilot/api/cluster/v1alpha1"
	projectv1alpha1 "github.com/f-rambo/cloud-copilot/api/project/v1alpha1"
	servicev1alpha1 "github.com/f-rambo/cloud-copilot/api/service/v1alpha1"
	userv1alpha1 "github.com/f-rambo/cloud-copilot/api/user/v1alpha1"
	workspacev1alpha1 "github.com/f-rambo/cloud-copilot/api/workspace/v1alpha1"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/f-rambo/cloud-copilot/internal/interfaces"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/selector"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Bootstrap, cluster *interfaces.ClusterInterface, app *interfaces.AppInterface, services *interfaces.ServicesInterface, user *interfaces.UserInterface, workspace *interfaces.WorkspaceInterface, project *interfaces.ProjectInterface, logger log.Logger) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			selector.Server(NewAuthServer(user, c), BizContext(cluster, project, workspace)).Match(NewWhiteListMatcher()).Build(),
			recovery.Recovery(),
			metadata.Server(),
		),
	}
	cserver := c.Server
	netWork := cserver.GetGrpc().GetNetwork()
	if netWork != "" {
		opts = append(opts, grpc.Network(netWork))
	}
	addr := cserver.GetGrpc().GetAddr()
	if addr != "" {
		opts = append(opts, grpc.Address(addr))
	}
	timeoutSecond := cserver.GetGrpc().GetTimeout()
	if timeoutSecond != 0 {
		opts = append(opts, grpc.Timeout(time.Duration(timeoutSecond)*time.Second))
	}
	srv := grpc.NewServer(opts...)
	clusterv1alpha1.RegisterClusterInterfaceServer(srv, cluster)
	appv1alpha1.RegisterAppInterfaceServer(srv, app)
	servicev1alpha1.RegisterServiceInterfaceServer(srv, services)
	userv1alpha1.RegisterUserInterfaceServer(srv, user)
	workspacev1alpha1.RegisterWorkspaceInterfaceServer(srv, workspace)
	projectv1alpha1.RegisterProjectServiceServer(srv, project)
	return srv
}

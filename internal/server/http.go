package server

import (
	"strings"

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
	"github.com/go-kratos/kratos/v2/middleware/validate"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/gorilla/handlers"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Bootstrap, cluster *interfaces.ClusterInterface, app *interfaces.AppInterface, services *interfaces.ServicesInterface, user *interfaces.UserInterface, workspace *interfaces.WorkspaceInterface, project *interfaces.ProjectInterface, logger log.Logger) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			selector.Server(NewAuthServer(user, c), BizContext(cluster, project, workspace)).Match(NewWhiteListMatcher()).Build(),
			recovery.Recovery(),
			metadata.Server(),
			validate.Validator(),
		),
		http.RequestDecoder(func(r *http.Request, v any) error {
			if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
				return nil
			}
			return http.DefaultRequestDecoder(r, v)
		}),
		http.ResponseEncoder(http.DefaultResponseEncoder),
	}
	cserver := c.Server
	netWork := cserver.GetHttp().GetNetwork()
	if netWork != "" {
		opts = append(opts, http.Network(netWork))
	}
	addr := cserver.GetHttp().GetAddr()
	if addr != "" {
		opts = append(opts, http.Address(addr))
	}
	opts = append(opts, http.Filter(handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization", "Content-Length", "Content-Disposition"}),
	)))

	srv := http.NewServer(opts...)
	clusterv1alpha1.RegisterClusterInterfaceHTTPServer(srv, cluster)
	appv1alpha1.RegisterAppInterfaceHTTPServer(srv, app)
	servicev1alpha1.RegisterServiceInterfaceHTTPServer(srv, services)
	userv1alpha1.RegisterUserInterfaceHTTPServer(srv, user)
	workspacev1alpha1.RegisterWorkspaceInterfaceHTTPServer(srv, workspace)
	projectv1alpha1.RegisterProjectServiceHTTPServer(srv, project)
	return srv
}

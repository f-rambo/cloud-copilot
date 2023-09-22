package server

import (
	appv1alpha1 "github.com/f-rambo/ocean/api/app/v1alpha1"
	clusterv1alpha1 "github.com/f-rambo/ocean/api/cluster/v1alpha1"
	servicev1alpha1 "github.com/f-rambo/ocean/api/service/v1alpha1"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, cluster *service.ClusterService, app *service.AppService, services *service.ServicesService, logger log.Logger) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
		),
	}
	netWork := c.HTTP.GetNetwork()
	if netWork != "" {
		opts = append(opts, http.Network(netWork))
	}
	addr := c.HTTP.GetAddr()
	if addr != "" {
		opts = append(opts, http.Address(addr))
	}
	srv := http.NewServer(opts...)
	clusterv1alpha1.RegisterClusterServiceHTTPServer(srv, cluster)
	appv1alpha1.RegisterAppServiceHTTPServer(srv, app)
	servicev1alpha1.RegisterServiceServiceHTTPServer(srv, services)
	return srv
}

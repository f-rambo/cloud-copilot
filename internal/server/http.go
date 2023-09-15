package server

import (
	appv1 "github.com/f-rambo/ocean/api/app/v1"
	clusterv1 "github.com/f-rambo/ocean/api/cluster/v1"
	servicev1 "github.com/f-rambo/ocean/api/service/v1"
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
	clusterv1.RegisterClusterServiceHTTPServer(srv, cluster)
	appv1.RegisterAppServiceHTTPServer(srv, app)
	servicev1.RegisterServiceServiceHTTPServer(srv, services)
	return srv
}

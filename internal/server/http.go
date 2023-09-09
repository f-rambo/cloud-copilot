package server

import (
	appv1 "ocean/api/app/v1"
	clusterv1 "ocean/api/cluster/v1"
	helloworldv1 "ocean/api/helloworld/v1"
	"ocean/internal/conf"
	"ocean/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, greeter *service.GreeterService, cluster *service.ClusterService, app *service.AppService, logger log.Logger) *http.Server {
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
	helloworldv1.RegisterGreeterHTTPServer(srv, greeter)
	clusterv1.RegisterClusterServiceHTTPServer(srv, cluster)
	appv1.RegisterAppServiceHTTPServer(srv, app)
	return srv
}

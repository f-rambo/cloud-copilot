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
	"github.com/go-kratos/kratos/v2/transport/http"
)

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, greeter *service.GreeterService, infra *service.InfraService,
	cluster *service.ClusterService, app *service.AppService, logger log.Logger) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
		),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, http.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	helloworldv1.RegisterGreeterHTTPServer(srv, greeter)
	infrav1.RegisterInfraHTTPServer(srv, infra)
	clusterv1.RegisterClusterHTTPServer(srv, cluster)
	appv1.RegisterAppServiceHTTPServer(srv, app)
	return srv
}

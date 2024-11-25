package clusterruntime

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	userApi "github.com/f-rambo/cloud-copilot/internal/repository/clusterruntime/api/user"
	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/go-kratos/kratos/v2/log"
)

type ClusterRuntimeUser struct {
	conf *conf.Bootstrap
	log  *log.Helper
}

func NewClusterRuntimeUser(conf *conf.Bootstrap, logger log.Logger) biz.Thirdparty {
	return &ClusterRuntimeUser{
		conf: conf,
		log:  log.NewHelper(logger),
	}
}

func (u *ClusterRuntimeUser) getServiceConfig() *conf.Service {
	for _, service := range u.conf.Services {
		if service.Name == ServiceNameClusterRuntime {
			return service
		}
	}
	return nil
}

func (u *ClusterRuntimeUser) GetUserEmail(ctx context.Context, token string) (string, error) {
	service := u.getServiceConfig()
	grpconn, err := new(utils.GrpcConn).OpenGrpcConn(ctx, service.Addr, service.Port)
	if err != nil {
		return "", err
	}
	defer grpconn.Close()
	res, err := userApi.NewUserInterfaceClient(grpconn.Conn).GetUserEmail(ctx, &userApi.UserEmaileReq{
		Token: token,
	})
	if err != nil {
		return "", err
	}
	if res == nil {
		return "", nil
	}
	return res.Email, nil
}

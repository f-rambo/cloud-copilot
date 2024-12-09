package clusterruntime

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	userApi "github.com/f-rambo/cloud-copilot/internal/repository/clusterruntime/api/user"
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

func (u *ClusterRuntimeUser) GetUserEmail(ctx context.Context, token string) (string, error) {
	grpconn, err := connGrpc(ctx, u.conf)
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

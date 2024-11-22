package clusterruntime

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
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
	return "", nil
}

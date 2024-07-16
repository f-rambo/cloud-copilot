package kubernetes

import (
	"context"

	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
)

type ClusterRuntime struct {
	log *log.Helper
	c   *conf.Bootstrap
}

func NewClusterRuntime(c *conf.Bootstrap, logger log.Logger) biz.ClusterRuntime {
	return &ClusterRuntime{
		log: log.NewHelper(logger),
		c:   c,
	}
}

func (cr *ClusterRuntime) CurrentCluster(ctx context.Context) (*biz.Cluster, error) {
	// TODO: 实现当前集群获取
	return nil, nil
}

func (cr *ClusterRuntime) ConnectCluster(ctx context.Context, cluster *biz.Cluster) error {
	// TODO: 实现集群连接
	return nil
}

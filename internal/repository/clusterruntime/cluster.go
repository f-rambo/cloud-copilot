package clusterruntime

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
)

type ClusterRuntimeCluster struct {
	conf *conf.Bootstrap
	log  *log.Helper
}

func NewClusterRuntimeCluster(conf *conf.Bootstrap, logger log.Logger) biz.ClusterRuntime {
	return &ClusterRuntimeCluster{
		conf: conf,
		log:  log.NewHelper(logger),
	}
}

func (c *ClusterRuntimeCluster) CurrentCluster(ctx context.Context, cluster *biz.Cluster) error {
	return nil
}

func (c *ClusterRuntimeCluster) HandlerNodes(ctx context.Context, cluster *biz.Cluster) error {
	return nil
}

func (c *ClusterRuntimeCluster) MigrateToCluster(ctx context.Context, cluster *biz.Cluster) error {
	return nil
}

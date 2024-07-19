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

func (cr *ClusterRuntime) CurrentCluster(ctx context.Context, cluster *biz.Cluster) error {
	_, err := getKubeClientSet()
	if err != nil {
		return err
	}
	return nil
}

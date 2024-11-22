package clusterruntime

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
)

type ClusterRuntimeProject struct {
	conf *conf.Bootstrap
	log  *log.Helper
}

func NewClusterRuntimeProject(conf *conf.Bootstrap, logger log.Logger) biz.ProjectRuntime {
	return &ClusterRuntimeProject{
		conf: conf,
		log:  log.NewHelper(logger),
	}
}

func (c *ClusterRuntimeProject) CreateNamespace(ctx context.Context, namespace string) error {
	return nil
}

func (c *ClusterRuntimeProject) GetNamespaces(ctx context.Context) (namespaces []string, err error) {
	return nil, nil
}

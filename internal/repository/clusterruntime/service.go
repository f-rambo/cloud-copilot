package clusterruntime

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
)

type ClusterRuntimeService struct {
	conf *conf.Bootstrap
	log  *log.Helper
}

func NewClusterRuntimeService(conf *conf.Bootstrap, logger log.Logger) biz.WorkflowRuntime {
	return &ClusterRuntimeService{
		conf: conf,
		log:  log.NewHelper(logger),
	}
}

func (s *ClusterRuntimeService) CommitWorklfow(ctx context.Context, wf *biz.Workflow) error {
	return nil
}

func (s *ClusterRuntimeService) GetWorkflow(ctx context.Context, wf *biz.Workflow) error {
	return nil
}

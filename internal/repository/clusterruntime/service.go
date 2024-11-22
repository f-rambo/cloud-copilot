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

func (s *ClusterRuntimeService) GenerateCIWorkflow(ctx context.Context, service *biz.Service) (ciWf *biz.Workflow, cdwf *biz.Workflow, err error) {
	return nil, nil, nil
}

func (s *ClusterRuntimeService) Create(ctx context.Context, namespace string, workflow *biz.Workflow) error {
	return nil
}

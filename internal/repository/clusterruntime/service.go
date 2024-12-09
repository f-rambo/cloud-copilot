package clusterruntime

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	serviceApi "github.com/f-rambo/cloud-copilot/internal/repository/clusterruntime/api/service"
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
	grpconn, err := connGrpc(ctx, s.conf)
	if err != nil {
		return nil, nil, err
	}
	defer grpconn.Close()
	res, err := serviceApi.NewServiceInterfaceClient(grpconn.Conn).GenerateCIWorkflow(ctx, service)
	if err != nil {
		return nil, nil, err
	}
	return res.CiWorkflow, res.CdWorkflow, nil
}

func (s *ClusterRuntimeService) Create(ctx context.Context, namespace string, workflow *biz.Workflow) error {
	grpconn, err := connGrpc(ctx, s.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	_, err = serviceApi.NewServiceInterfaceClient(grpconn.Conn).Create(ctx, &serviceApi.CreateReq{
		Namespace: namespace,
		Workflow:  workflow,
	})
	if err != nil {
		return err
	}
	return nil
}

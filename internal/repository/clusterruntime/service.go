package clusterruntime

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	serviceApi "github.com/f-rambo/cloud-copilot/internal/repository/clusterruntime/api/service"
	"github.com/f-rambo/cloud-copilot/utils"
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

func (c *ClusterRuntimeService) getClusterRuntimeServiceServiceConfig() *conf.Service {
	for _, service := range c.conf.Services {
		if service.Name == ServiceNameClusterRuntime {
			return service
		}
	}
	return nil
}

func (s *ClusterRuntimeService) GenerateCIWorkflow(ctx context.Context, service *biz.Service) (ciWf *biz.Workflow, cdwf *biz.Workflow, err error) {
	serviceConf := s.getClusterRuntimeServiceServiceConfig()
	grpconn, err := new(utils.GrpcConn).OpenGrpcConn(ctx, serviceConf.Addr, serviceConf.Port)
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
	serviceConf := s.getClusterRuntimeServiceServiceConfig()
	grpconn, err := new(utils.GrpcConn).OpenGrpcConn(ctx, serviceConf.Addr, serviceConf.Port)
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

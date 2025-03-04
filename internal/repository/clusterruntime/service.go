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

func NewClusterRuntimeService(conf *conf.Bootstrap, logger log.Logger) biz.ServiceRuntime {
	return &ClusterRuntimeService{
		conf: conf,
		log:  log.NewHelper(logger),
	}
}

func (s *ClusterRuntimeService) ApplyService(ctx context.Context, service *biz.Service, cd *biz.ContinuousDeployment) error {
	grpconn, err := connGrpc(ctx, s.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	_, err = serviceApi.NewServiceInterfaceClient(grpconn.Conn).ApplyService(ctx, &serviceApi.ApplyServiceRequest{Service: service, Cd: cd})
	if err != nil {
		return err
	}
	return nil
}

func (s *ClusterRuntimeService) GetService(ctx context.Context, service *biz.Service) error {
	grpconn, err := connGrpc(ctx, s.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	serviceRes, err := serviceApi.NewServiceInterfaceClient(grpconn.Conn).GetService(ctx, service)
	if err != nil {
		return err
	}
	err = utils.StructTransform(serviceRes, service)
	if err != nil {
		return err
	}
	return nil
}

func (s *ClusterRuntimeService) CommitWorkflow(ctx context.Context, wf *biz.Workflow) error {
	grpconn, err := connGrpc(ctx, s.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	_, err = serviceApi.NewServiceInterfaceClient(grpconn.Conn).CommitWorkflow(ctx, wf)
	if err != nil {
		return err
	}
	return nil
}

func (s *ClusterRuntimeService) GetWorkflow(ctx context.Context, wf *biz.Workflow) error {
	grpconn, err := connGrpc(ctx, s.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	wfRes, err := serviceApi.NewServiceInterfaceClient(grpconn.Conn).GetWorkflow(ctx, wf)
	if err != nil {
		return err
	}
	err = utils.StructTransform(wfRes, wf)
	if err != nil {
		return err
	}
	return nil
}

func (s *ClusterRuntimeService) CleanWorkflow(ctx context.Context, wf *biz.Workflow) error {
	grpconn, err := connGrpc(ctx, s.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	_, err = serviceApi.NewServiceInterfaceClient(grpconn.Conn).CleanWorkflow(ctx, wf)
	if err != nil {
		return err
	}
	return nil
}

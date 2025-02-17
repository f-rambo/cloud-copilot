package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
)

type ServicesData interface {
	List(ctx context.Context, serviceParam *Service, page, pageSize int) ([]*Service, int64, error)
	Save(ctx context.Context, service *Service) error
	Get(ctx context.Context, id int64) (*Service, error)
	Delete(ctx context.Context, id int64) error
	GetWorkflow(ctx context.Context, id int64) (*Workflow, error)
	SaveWrkflow(ctx context.Context, workflow *Workflow) error
	DeleteWrkflow(ctx context.Context, id int64) error
	GetServiceCis(ctx context.Context, serviceId int64, page, pageSize int32) ([]*ContinuousIntegration, int64, error)
}

type WorkflowRuntime interface {
	GenerateCIWorkflow(context.Context, *Service) (ciWf *Workflow, cdwf *Workflow, err error)
	Create(ctx context.Context, namespace string, workflow *Workflow) error
}

type ServiceAgent interface {
}

type ServicesUseCase struct {
	serviceData     ServicesData
	workflowRuntime WorkflowRuntime
	log             *log.Helper
}

func NewServicesUseCase(serviceData ServicesData, wfRuntime WorkflowRuntime, logger log.Logger) *ServicesUseCase {
	return &ServicesUseCase{serviceData: serviceData, workflowRuntime: wfRuntime, log: log.NewHelper(logger)}
}

func (uc *ServicesUseCase) List(ctx context.Context, serviceParam *Service, page, pageSize int) ([]*Service, int64, error) {
	return uc.serviceData.List(ctx, serviceParam, page, pageSize)
}

func (uc *ServicesUseCase) Save(ctx context.Context, service *Service) error {
	if service.Id == 0 {
		ciWf, cdWf, err := uc.workflowRuntime.GenerateCIWorkflow(ctx, service)
		if err != nil {
			return err
		}
		err = uc.serviceData.SaveWrkflow(ctx, ciWf)
		if err != nil {
			return err
		}
		service.CiWorkflowId = ciWf.Id
		err = uc.serviceData.SaveWrkflow(ctx, cdWf)
		if err != nil {
			return err
		}
		service.CdWorkflowId = cdWf.Id
	}
	return uc.serviceData.Save(ctx, service)
}

func (uc *ServicesUseCase) Get(ctx context.Context, id int64) (*Service, error) {
	return uc.serviceData.Get(ctx, id)
}

func (uc *ServicesUseCase) Delete(ctx context.Context, id int64) error {
	return uc.serviceData.Delete(ctx, id)
}

func (uc *ServicesUseCase) GetWorkflow(ctx context.Context, id int64, wfType WorkflowType) (*Workflow, error) {
	service, err := uc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	wf, err := uc.serviceData.GetWorkflow(ctx, service.CiWorkflowId)
	if err != nil {
		return nil, err
	}
	return wf, nil
}

func (uc *ServicesUseCase) SaveWorkflow(ctx context.Context, serviceId int64, wfType WorkflowType, wf *Workflow) error {
	service, err := uc.Get(ctx, serviceId)
	if err != nil {
		return err
	}
	return uc.serviceData.Save(ctx, service)
}

func (uc *ServicesUseCase) CommitWorklfow(ctx context.Context, project *Project, service *Service, wfType WorkflowType, workflowsId int64) error {
	wf, err := uc.serviceData.GetWorkflow(ctx, workflowsId)
	if err != nil {
		return err
	}
	return uc.workflowRuntime.Create(ctx, project.Name, wf)
}

func (uc *ServicesUseCase) GetServiceCis(ctx context.Context, serviceId int64, page, pageSize int32) ([]*ContinuousIntegration, int64, error) {
	return uc.serviceData.GetServiceCis(ctx, serviceId, page, pageSize)
}

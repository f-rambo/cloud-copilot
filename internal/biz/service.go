package biz

import (
	"context"
	"encoding/json"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
)

type ServicesData interface {
	Save(ctx context.Context, service *Service) error
	Get(ctx context.Context, id int64) (*Service, error)
	List(ctx context.Context, projectId int64, serviceName string, page, pageSize int32) ([]*Service, int64, error)
	Delete(ctx context.Context, id int64) error
	GetServiceResourceByProject(ctx context.Context, projectId int64, alreadyResource *AlreadyResource) error
	GetByName(ctx context.Context, projectId int64, name string) (*Service, error)
	SaveWorkflow(ctx context.Context, workflow *Workflow) error
	GetWorkflowByServiceId(ctx context.Context, serviceId int64) ([]*Workflow, error)
	SaveContinuousIntegration(context.Context, *ContinuousIntegration) error
	GetContinuousIntegration(context.Context, int64) (*ContinuousIntegration, error)
	DeleteContinuousIntegration(context.Context, int64) error
	GetContinuousIntegrations(ctx context.Context, serviceId int64, page, pageSize int32) ([]*ContinuousIntegration, int64, error)
	SaveContinuousDeployment(context.Context, *ContinuousDeployment) error
	GetContinuousDeployment(context.Context, int64) (*ContinuousDeployment, error)
	DeleteContinuousDeployment(context.Context, int64) error
	GetContinuousDeployments(ctx context.Context, serviceId int64, page, pageSize int32) ([]*ContinuousDeployment, int64, error)
}

type WorkflowRuntime interface {
	CommitWorklfow(context.Context, *Workflow) error
	GetWorkflow(context.Context, *Workflow) error
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

func (w *Workflow) SettingContinuousIntegration(service *Service, ci *ContinuousIntegration) error {
	return nil
}

func (w *Workflow) SettingContinuousDeployment(service *Service, ci *ContinuousIntegration, cd *ContinuousDeployment) error {
	return nil
}

func (ci *ContinuousIntegration) SetWorkflow(wf *Workflow) error {
	jsonData, err := json.Marshal(wf)
	if err != nil {
		return err
	}
	ci.WorkflowRuntime = string(jsonData)
	return nil
}

func (ci *ContinuousIntegration) GetWorkflow() (*Workflow, error) {
	if ci.WorkflowRuntime == "" {
		return nil, nil
	}
	var workflow Workflow
	err := json.Unmarshal([]byte(ci.WorkflowRuntime), &workflow)
	if err != nil {
		return nil, err
	}
	return &workflow, nil
}

func (cd *ContinuousDeployment) SetWorkflow(wf *Workflow) error {
	jsonData, err := json.Marshal(wf)
	if err != nil {
		return err
	}
	cd.WorkflowRuntime = string(jsonData)
	return nil
}

func (cd *ContinuousDeployment) GetWorkflow() (*Workflow, error) {
	if cd.WorkflowRuntime == "" {
		return nil, nil
	}
	var workflow Workflow
	err := json.Unmarshal([]byte(cd.WorkflowRuntime), &workflow)
	if err != nil {
		return nil, err
	}
	return &workflow, nil
}

func (uc *ServicesUseCase) Save(ctx context.Context, service *Service) error {
	if service.Id == 0 {
		serviceData, err := uc.serviceData.GetByName(ctx, service.ProjectId, service.Name)
		if err != nil {
			return err
		}
		if serviceData.Id > 0 {
			return errors.New("service name already exists")
		}
	}
	return uc.serviceData.Save(ctx, service)
}

func (uc *ServicesUseCase) Get(ctx context.Context, id int64) (*Service, error) {
	return uc.serviceData.Get(ctx, id)
}

func (uc *ServicesUseCase) List(ctx context.Context, projectId int64, serviceName string, page, pageSize int32) ([]*Service, int64, error) {
	return uc.serviceData.List(ctx, projectId, serviceName, page, pageSize)
}

func (uc *ServicesUseCase) Delete(ctx context.Context, id int64) error {
	return uc.serviceData.Delete(ctx, id)
}

func (uc *ServicesUseCase) GetServiceResourceByProject(ctx context.Context, projectId int64) (*AlreadyResource, error) {
	alreadyResource := &AlreadyResource{}
	err := uc.serviceData.GetServiceResourceByProject(ctx, projectId, alreadyResource)
	if err != nil {
		return nil, err
	}
	return alreadyResource, nil
}

func (uc *ServicesUseCase) SaveWorkflow(ctx context.Context, serviceId int64, wf *Workflow) error {
	service, err := uc.Get(ctx, serviceId)
	if err != nil {
		return err
	}
	if service.Id == 0 {
		return errors.New("service not found")
	}
	if wf.ServiceId == 0 {
		wf.ServiceId = serviceId
	}
	if wf.Id == 0 {
		workflows, err := uc.serviceData.GetWorkflowByServiceId(ctx, serviceId)
		if err != nil {
			return err
		}
		for _, v := range workflows {
			if v.Type == wf.Type {
				return errors.New("workflow already exists")
			}
		}
	}
	return uc.serviceData.SaveWorkflow(ctx, wf)
}

func (uc *ServicesUseCase) GetWorkflow(ctx context.Context, serviceId int64, wfType WorkflowType) (*Workflow, error) {
	workflows, err := uc.serviceData.GetWorkflowByServiceId(ctx, serviceId)
	if err != nil {
		return nil, err
	}
	for _, v := range workflows {
		if v.Type == wfType {
			return v, nil
		}
	}
	return nil, errors.New("workflow not found")
}

type Workflows []*Workflow

func (ws Workflows) GetWorkflowByType(wfType WorkflowType) *Workflow {
	for _, v := range ws {
		if v.Type == wfType {
			return v
		}
	}
	return nil
}

func (uc *ServicesUseCase) CreateContinuousIntegration(ctx context.Context, ci *ContinuousIntegration) error {
	service, err := uc.Get(ctx, ci.ServiceId)
	if err != nil {
		return err
	}
	var workflows Workflows
	workflows, err = uc.serviceData.GetWorkflowByServiceId(ctx, service.Id)
	if err != nil {
		return err
	}
	workflow := workflows.GetWorkflowByType(WorkflowType_ContinuousIntegrationType)
	if workflow == nil {
		return errors.New("workflow not found")
	}
	err = workflow.SettingContinuousIntegration(service, ci)
	if err != nil {
		return err
	}
	err = uc.workflowRuntime.CommitWorklfow(ctx, workflow)
	if err != nil {
		return err
	}
	err = ci.SetWorkflow(workflow)
	if err != nil {
		return err
	}
	ci.Status = WorkfloStatus_Pending
	return uc.serviceData.SaveContinuousIntegration(ctx, ci)
}

func (uc *ServicesUseCase) GetContinuousIntegration(ctx context.Context, ciId int64) (*ContinuousIntegration, *Workflow, error) {
	ci, err := uc.serviceData.GetContinuousIntegration(ctx, ciId)
	if err != nil {
		return nil, nil, err
	}
	workflow, err := ci.GetWorkflow()
	if err != nil {
		return nil, nil, err
	}
	err = uc.workflowRuntime.GetWorkflow(ctx, workflow)
	if err != nil {
		return nil, nil, err
	}
	return ci, workflow, nil
}

func (uc *ServicesUseCase) UpdateContinuousIntegration(ctx context.Context, ciId int64) error {
	ci, workflow, err := uc.GetContinuousIntegration(ctx, ciId)
	if err != nil {
		return err
	}
	err = ci.SetWorkflow(workflow)
	if err != nil {
		return err
	}
	defaultStatus := WorkfloStatus_Pending
	taskPendingNumber := 0
	for _, step := range workflow.WorkflowSteps {
		for _, task := range step.WorkflowTasks {
			if task.Status == WorkfloStatus_Failure {
				defaultStatus = WorkfloStatus_Failure
				break
			}
			if task.Status == WorkfloStatus_Pending {
				taskPendingNumber++
			}
		}
	}
	if defaultStatus == WorkfloStatus_Failure {
		ci.Status = WorkfloStatus_Failure
	}
	if defaultStatus != WorkfloStatus_Failure && taskPendingNumber == 0 {
		ci.Status = WorkfloStatus_Success
	}
	return uc.serviceData.SaveContinuousIntegration(ctx, ci)
}

func (uc *ServicesUseCase) GetContinuousIntegrations(ctx context.Context, serviceId int64, page, pageSize int32) ([]*ContinuousIntegration, int64, error) {
	return uc.serviceData.GetContinuousIntegrations(ctx, serviceId, page, pageSize)
}

func (uc *ServicesUseCase) DeleteContinuousIntegration(ctx context.Context, ciId int64) error {
	return uc.serviceData.DeleteContinuousIntegration(ctx, ciId)
}

func (uc *ServicesUseCase) CreateContinuousDeployment(ctx context.Context, cd *ContinuousDeployment) error {
	service, err := uc.Get(ctx, cd.ServiceId)
	if err != nil {
		return err
	}
	ci, err := uc.serviceData.GetContinuousIntegration(ctx, cd.CiId)
	if err != nil {
		return err
	}
	var workflows Workflows
	workflows, err = uc.serviceData.GetWorkflowByServiceId(ctx, service.Id)
	if err != nil {
		return err
	}
	workflow := workflows.GetWorkflowByType(WorkflowType_ContinuousDeploymentType)
	workflow.SettingContinuousDeployment(service, ci, cd)
	err = uc.workflowRuntime.CommitWorklfow(ctx, workflow)
	if err != nil {
		return err
	}
	err = cd.SetWorkflow(workflow)
	if err != nil {
		return err
	}
	cd.Status = WorkfloStatus_Pending
	return uc.serviceData.SaveContinuousDeployment(ctx, cd)
}

func (uc *ServicesUseCase) GetContinuousDeployment(ctx context.Context, cdId int64) (*ContinuousDeployment, *Workflow, error) {
	cd, err := uc.serviceData.GetContinuousDeployment(ctx, cdId)
	if err != nil {
		return nil, nil, err
	}
	workflow, err := cd.GetWorkflow()
	if err != nil {
		return nil, nil, err
	}
	err = uc.workflowRuntime.GetWorkflow(ctx, workflow)
	if err != nil {
		return nil, nil, err
	}
	return cd, workflow, nil
}

func (uc *ServicesUseCase) UpdateContinuousDeployment(ctx context.Context, cdId int64) error {
	cd, workflow, err := uc.GetContinuousDeployment(ctx, cdId)
	if err != nil {
		return err
	}
	err = cd.SetWorkflow(workflow)
	if err != nil {
		return err
	}
	defaultStatus := WorkfloStatus_Pending
	taskPendingNumber := 0
	for _, step := range workflow.WorkflowSteps {
		for _, task := range step.WorkflowTasks {
			if task.Status == WorkfloStatus_Failure {
				defaultStatus = WorkfloStatus_Failure
				break
			}
			if task.Status == WorkfloStatus_Pending {
				taskPendingNumber++
			}
		}
	}
	if defaultStatus == WorkfloStatus_Failure {
		cd.Status = WorkfloStatus_Failure
	}
	if defaultStatus != WorkfloStatus_Failure && taskPendingNumber == 0 {
		cd.Status = WorkfloStatus_Success
	}
	return uc.serviceData.SaveContinuousDeployment(ctx, cd)
}

func (uc *ServicesUseCase) GetContinuousDeployments(ctx context.Context, serviceId int64, page, pageSize int32) ([]*ContinuousDeployment, int64, error) {
	return uc.serviceData.GetContinuousDeployments(ctx, serviceId, page, pageSize)
}

func (uc *ServicesUseCase) DeleteContinuousDeployment(ctx context.Context, cdId int64) error {
	return uc.serviceData.DeleteContinuousDeployment(ctx, cdId)
}

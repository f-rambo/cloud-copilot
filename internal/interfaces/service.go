package interfaces

import (
	"context"
	"encoding/json"

	"github.com/f-rambo/cloud-copilot/api/common"
	v1alpha1 "github.com/f-rambo/cloud-copilot/api/service/v1alpha1"
	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/pkg/errors"
)

type ServicesInterface struct {
	v1alpha1.UnimplementedServiceInterfaceServer
	serviceUc *biz.ServicesUseCase
}

func NewServicesInterface(serviceUc *biz.ServicesUseCase) *ServicesInterface {
	return &ServicesInterface{serviceUc: serviceUc}
}

func (s *ServicesInterface) List(ctx context.Context, serviceReq *v1alpha1.ServicesRequest) (*v1alpha1.Services, error) {
	if serviceReq.ProjectId == 0 {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	serviceData, total, err := s.serviceUc.List(ctx, int64(serviceReq.ProjectId), serviceReq.Name, serviceReq.Page, serviceReq.Size)
	if err != nil {
		return nil, err
	}
	res := &v1alpha1.Services{}
	res.Total = int32(total)
	for _, v := range serviceData {
		res.Services = append(res.Services, s.serviceBizTointerface(v))
	}
	return res, nil
}

func (s *ServicesInterface) Save(ctx context.Context, service *v1alpha1.Service) (*common.Msg, error) {
	if service.ProjectId == 0 || service.Name == "" {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	if !utils.IsValidKubernetesName(service.Name) {
		return nil, errors.New("service name is invalid")
	}
	err := s.serviceUc.Save(ctx, s.serviceInterfaceToBiz(service))
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (s *ServicesInterface) Get(ctx context.Context, serviceReq *v1alpha1.ServiceDetailIdRequest) (*v1alpha1.Service, error) {
	if serviceReq.Id == 0 {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	service, err := s.serviceUc.Get(ctx, int64(serviceReq.Id))
	if err != nil {
		return nil, err
	}
	return s.serviceBizTointerface(service), nil
}

func (s *ServicesInterface) Delete(ctx context.Context, serviceReq *v1alpha1.ServiceDetailIdRequest) (*common.Msg, error) {
	if serviceReq.Id == 0 {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	err := s.serviceUc.Delete(ctx, int64(serviceReq.Id))
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (s *ServicesInterface) SaveServiceWorkflow(ctx context.Context, wf *v1alpha1.Workflow) (*common.Msg, error) {
	if wf.ServiceId == 0 || wf.Name == "" {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	if !utils.IsValidKubernetesName(wf.Name) {
		return nil, errors.New("workflow name is invalid")
	}
	err := s.serviceUc.SaveWorkflow(ctx, int64(wf.ServiceId), s.workflowInterfaceToBiz(wf))
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (s *ServicesInterface) GetWorkflow(ctx context.Context, wfReq *v1alpha1.GetServiceWorkflowRequest) (*v1alpha1.Workflow, error) {
	if wfReq.ServiceId == 0 || wfReq.WorkflowType == "" {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	wfType := biz.WorkflowTypeFindByString(wfReq.WorkflowType)
	if wfType == biz.WorkflowType_UNSPECIFIED {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	workflow, err := s.serviceUc.GetWorkflow(ctx, int64(wfReq.ServiceId), biz.WorkflowType(wfType))
	if err != nil {
		return nil, err
	}
	if workflow == nil || len(workflow.WorkflowSteps) == 0 {
		workflow, err = s.serviceUc.GetDefaultWorkflow(ctx, int64(wfReq.ServiceId), biz.WorkflowType(wfType))
		if err != nil {
			return nil, err
		}
	}
	return s.workflowBizToInterface(workflow), nil
}

func (s *ServicesInterface) CreateContinuousIntegration(ctx context.Context, ci *v1alpha1.ContinuousIntegration) (*common.Msg, error) {
	if ci.ServiceId == 0 {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	err := s.serviceUc.CreateContinuousIntegration(ctx, s.ciInterfaceToBiz(ci))
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (s *ServicesInterface) GetContinuousIntegration(ctx context.Context, ciReq *v1alpha1.ContinuousIntegrationDetailRequest) (*v1alpha1.ContinuousIntegration, error) {
	if ciReq.Id == 0 {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	ci, err := s.serviceUc.GetContinuousIntegration(ctx, int64(ciReq.Id))
	if err != nil {
		return nil, err
	}
	return s.ciBizToInterface(ci), nil
}

func (s *ServicesInterface) GetContinuousIntegrations(ctx context.Context, ciReq *v1alpha1.ContinuousIntegrationsRequest) (*v1alpha1.ContinuousIntegrations, error) {
	if ciReq.ServiceId == 0 {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	ciData, total, err := s.serviceUc.GetContinuousIntegrations(ctx, int64(ciReq.ServiceId), ciReq.Page, ciReq.PageSize)
	if err != nil {
		return nil, err
	}
	ciRes := &v1alpha1.ContinuousIntegrations{}
	ciRes.Total = int32(total)
	for _, v := range ciData {
		ciRes.ContinuousIntegrations = append(ciRes.ContinuousIntegrations, s.ciBizToInterface(v))
	}
	return ciRes, nil
}

func (s *ServicesInterface) DeleteContinuousIntegration(ctx context.Context, ciReq *v1alpha1.ContinuousIntegrationDetailRequest) (*common.Msg, error) {
	if ciReq.Id == 0 {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	err := s.serviceUc.DeleteContinuousIntegration(ctx, int64(ciReq.Id))
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (s *ServicesInterface) CreateContinuousDeployment(ctx context.Context, cd *v1alpha1.ContinuousDeployment) (*common.Msg, error) {
	if cd.ServiceId == 0 || cd.Config == "" || cd.ConfigPath == "" {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	err := s.serviceUc.CreateContinuousDeployment(ctx, s.cdInterfaceToBiz(cd))
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (s *ServicesInterface) GetContinuousDeployment(ctx context.Context, cdReq *v1alpha1.ContinuousDeploymentDetailRequest) (*v1alpha1.ContinuousDeployment, error) {
	if cdReq.Id == 0 {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	cd, err := s.serviceUc.GetContinuousDeployment(ctx, int64(cdReq.Id))
	if err != nil {
		return nil, err
	}
	return s.cdBizToInterface(cd), nil
}

func (s *ServicesInterface) GetContinuousDeployments(ctx context.Context, cdReq *v1alpha1.ContinuousDeploymentsRequest) (*v1alpha1.ContinuousDeployments, error) {
	if cdReq.ServiceId == 0 {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	cdData, total, err := s.serviceUc.GetContinuousDeployments(ctx, int64(cdReq.ServiceId), cdReq.Page, cdReq.PageSize)
	if err != nil {
		return nil, err
	}
	cdRes := &v1alpha1.ContinuousDeployments{}
	cdRes.Total = int32(total)
	for _, v := range cdData {
		cdRes.ContinuousDeployments = append(cdRes.ContinuousDeployments, s.cdBizToInterface(v))
	}
	return cdRes, nil
}

func (s *ServicesInterface) DeleteContinuousDeployment(ctx context.Context, cdReq *v1alpha1.ContinuousDeploymentDetailRequest) (*common.Msg, error) {
	if cdReq.Id == 0 {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	err := s.serviceUc.DeleteContinuousDeployment(ctx, int64(cdReq.Id))
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (s *ServicesInterface) ApplyService(ctx context.Context, serviceReq *v1alpha1.ApplyServiceRequest) (*common.Msg, error) {
	if serviceReq.ServiceId == 0 || serviceReq.CiId == 0 || serviceReq.CdId == 0 {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	err := s.serviceUc.ApplyService(ctx, int64(serviceReq.ServiceId), int64(serviceReq.CiId), int64(serviceReq.CdId))
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (s *ServicesInterface) serviceBizTointerface(bizService *biz.Service) *v1alpha1.Service {
	if bizService == nil {
		return nil
	}
	result := &v1alpha1.Service{
		Id:            int32(bizService.Id),
		Name:          bizService.Name,
		Labels:        bizService.Lables,
		Description:   bizService.Description,
		UserId:        int32(bizService.UserId),
		ProjectId:     int32(bizService.ProjectId),
		WorkspaceId:   int32(bizService.WorkspaceId),
		ClusterId:     int32(bizService.ClusterId),
		ResourceQuota: resourceQuotaBizToInterface(bizService.ResourceQuota),
	}

	// 转换Ports
	if bizService.Ports != nil {
		result.Ports = make([]*v1alpha1.Port, 0, len(bizService.Ports))
		for _, port := range bizService.Ports {
			result.Ports = append(result.Ports, &v1alpha1.Port{
				Id:            int32(port.Id),
				Name:          port.Name,
				Path:          port.Path,
				Protocol:      port.Protocol,
				ContainerPort: port.ContainerPort,
			})
		}
	}

	// 转换Volumes
	if bizService.Volumes != nil {
		result.Volumes = make([]*v1alpha1.Volume, 0, len(bizService.Volumes))
		for _, volume := range bizService.Volumes {
			result.Volumes = append(result.Volumes, &v1alpha1.Volume{
				Id:           int32(volume.Id),
				Name:         volume.Name,
				MountPath:    volume.MountPath,
				Storage:      volume.Storage,
				StorageClass: volume.StorageClass,
			})
		}
	}

	// 转换Pods
	if bizService.Pods != nil {
		result.Pods = make([]*v1alpha1.Pod, 0, len(bizService.Pods))
		for _, pod := range bizService.Pods {
			result.Pods = append(result.Pods, &v1alpha1.Pod{
				Id:       int32(pod.Id),
				Name:     pod.Name,
				NodeName: pod.NodeName,
				Status:   pod.Status.String(),
			})
		}
	}

	return result
}

func (s *ServicesInterface) serviceInterfaceToBiz(interfaceServer *v1alpha1.Service) *biz.Service {
	if interfaceServer == nil {
		return nil
	}
	result := &biz.Service{
		Id:            int64(interfaceServer.Id),
		Name:          interfaceServer.Name,
		Lables:        interfaceServer.Labels,
		Description:   interfaceServer.Description,
		UserId:        int64(interfaceServer.UserId),
		ProjectId:     int64(interfaceServer.ProjectId),
		WorkspaceId:   int64(interfaceServer.WorkspaceId),
		ClusterId:     int64(interfaceServer.ClusterId),
		ResourceQuota: resourceQuotaInterfaceToBiz(interfaceServer.ResourceQuota),
	}

	// 转换Ports
	if interfaceServer.Ports != nil {
		result.Ports = make([]*biz.Port, 0, len(interfaceServer.Ports))
		for _, port := range interfaceServer.Ports {
			result.Ports = append(result.Ports, &biz.Port{
				Id:            int64(port.Id),
				Name:          port.Name,
				Path:          port.Path,
				Protocol:      port.Protocol,
				ContainerPort: port.ContainerPort,
				ServiceId:     int64(interfaceServer.Id),
			})
		}
	}

	// 转换Volumes
	if interfaceServer.Volumes != nil {
		result.Volumes = make([]*biz.Volume, 0, len(interfaceServer.Volumes))
		for _, volume := range interfaceServer.Volumes {
			result.Volumes = append(result.Volumes, &biz.Volume{
				Id:           int64(volume.Id),
				Name:         volume.Name,
				MountPath:    volume.MountPath,
				Storage:      volume.Storage,
				StorageClass: volume.StorageClass,
				ServiceId:    int64(interfaceServer.Id),
			})
		}
	}

	// 转换Pods
	if interfaceServer.Pods != nil {
		result.Pods = make([]*biz.Pod, 0, len(interfaceServer.Pods))
		for _, pod := range interfaceServer.Pods {
			result.Pods = append(result.Pods, &biz.Pod{
				Id:        int64(pod.Id),
				Name:      pod.Name,
				NodeName:  pod.NodeName,
				ServiceId: int64(interfaceServer.Id),
			})
		}
	}

	return result
}

func (s *ServicesInterface) workflowBizToInterface(bizWf *biz.Workflow) *v1alpha1.Workflow {
	if bizWf == nil {
		return nil
	}
	result := &v1alpha1.Workflow{
		Id:           int32(bizWf.Id),
		Name:         bizWf.Name,
		Namespace:    bizWf.Namespace,
		WorkflowType: bizWf.Type.String(),
		Description:  bizWf.Description,
		ServiceId:    int32(bizWf.ServiceId),
	}

	// 转换WorkflowSteps
	if bizWf.WorkflowSteps != nil {
		result.WorkflowSteps = make([]*v1alpha1.WorkflowStep, 0, len(bizWf.WorkflowSteps))
		for _, step := range bizWf.WorkflowSteps {
			interfaceStep := &v1alpha1.WorkflowStep{
				Id:          int32(step.Id),
				WorkflowId:  int32(step.WorkflowId),
				Order:       step.Order,
				Name:        step.Name,
				Description: step.Description,
			}

			// 转换WorkflowTasks
			if step.WorkflowTasks != nil {
				interfaceStep.WorkflowTasks = make([]*v1alpha1.WorkflowTask, 0, len(step.WorkflowTasks))
				for _, task := range step.WorkflowTasks {
					interfaceStep.WorkflowTasks = append(interfaceStep.WorkflowTasks, &v1alpha1.WorkflowTask{
						Id:          int32(task.Id),
						WorkflowId:  int32(task.WorkflowId),
						StepId:      int32(task.StepId),
						Name:        task.Name,
						Order:       task.Order,
						Task:        task.TaskCommand,
						Description: task.Description,
						Status:      task.Status.String(),
					})
				}
			}

			result.WorkflowSteps = append(result.WorkflowSteps, interfaceStep)
		}
	}

	return result
}

func (s *ServicesInterface) workflowInterfaceToBiz(interfaceServer *v1alpha1.Workflow) *biz.Workflow {
	if interfaceServer == nil {
		return nil
	}
	result := &biz.Workflow{
		Id:          int64(interfaceServer.Id),
		Name:        interfaceServer.Name,
		Namespace:   interfaceServer.Namespace,
		Type:        biz.WorkflowTypeFindByString(interfaceServer.WorkflowType),
		Description: interfaceServer.Description,
		ServiceId:   int64(interfaceServer.ServiceId),
	}

	if interfaceServer.WorkflowSteps != nil {
		result.WorkflowSteps = make([]*biz.WorkflowStep, 0, len(interfaceServer.WorkflowSteps))
		for _, step := range interfaceServer.WorkflowSteps {
			bizStep := &biz.WorkflowStep{
				Id:          int64(step.Id),
				WorkflowId:  int64(step.WorkflowId),
				Order:       step.Order,
				Name:        step.Name,
				Description: step.Description,
			}

			if step.WorkflowTasks != nil {
				bizStep.WorkflowTasks = make([]*biz.WorkflowTask, 0, len(step.WorkflowTasks))
				for _, task := range step.WorkflowTasks {
					bizStep.WorkflowTasks = append(bizStep.WorkflowTasks, &biz.WorkflowTask{
						Id:          int64(task.Id),
						WorkflowId:  int64(task.WorkflowId),
						StepId:      int64(task.StepId),
						Name:        task.Name,
						Order:       task.Order,
						TaskCommand: task.Task,
						Description: task.Description,
					})
				}
			}

			result.WorkflowSteps = append(result.WorkflowSteps, bizStep)
		}
	}

	return result
}

// ci biz to interface
func (s *ServicesInterface) ciBizToInterface(bizCi *biz.ContinuousIntegration) *v1alpha1.ContinuousIntegration {
	if bizCi == nil {
		return nil
	}
	result := &v1alpha1.ContinuousIntegration{
		Id:          int32(bizCi.Id),
		Version:     bizCi.Version,
		Branch:      bizCi.Branch,
		Tag:         bizCi.Tag,
		Status:      bizCi.Status.String(),
		Description: bizCi.Description,
		ServiceId:   int32(bizCi.ServiceId),
		UserId:      int32(bizCi.UserId),
	}
	wf, err := bizCi.GetWorkflow()
	if err == nil {
		result.Workflow = s.workflowBizToInterface(wf)
	}
	return result
}

// ci interface to biz
func (s *ServicesInterface) ciInterfaceToBiz(interfaceServer *v1alpha1.ContinuousIntegration) *biz.ContinuousIntegration {
	if interfaceServer == nil {
		return nil
	}
	result := &biz.ContinuousIntegration{
		Id:          int64(interfaceServer.Id),
		Version:     interfaceServer.Version,
		Branch:      interfaceServer.Branch,
		Tag:         interfaceServer.Tag,
		Description: interfaceServer.Description,
		ServiceId:   int64(interfaceServer.ServiceId),
		UserId:      int64(interfaceServer.UserId),
	}
	return result
}

// cd biz to interface
func (s *ServicesInterface) cdBizToInterface(bizCd *biz.ContinuousDeployment) *v1alpha1.ContinuousDeployment {
	if bizCd == nil {
		return nil
	}
	result := &v1alpha1.ContinuousDeployment{
		Id:        int32(bizCd.Id),
		CiId:      int32(bizCd.CiId),
		ServiceId: int32(bizCd.ServiceId),
		UserId:    int32(bizCd.UserId),
		Status:    bizCd.Status.String(),
	}
	if bizCd.Config != nil {
		configJSON, _ := json.Marshal(bizCd.Config)
		result.Config = string(configJSON)
	}
	wf, err := bizCd.GetWorkflow()
	if err == nil {
		result.Workflow = s.workflowBizToInterface(wf)
	}
	return result
}

// cd interface to biz
func (s *ServicesInterface) cdInterfaceToBiz(interfaceServer *v1alpha1.ContinuousDeployment) *biz.ContinuousDeployment {
	if interfaceServer == nil {
		return nil
	}
	result := &biz.ContinuousDeployment{
		Id:        int64(interfaceServer.Id),
		CiId:      int64(interfaceServer.CiId),
		ServiceId: int64(interfaceServer.ServiceId),
		UserId:    int64(interfaceServer.UserId),
	}
	if interfaceServer.Config != "" {
		config := make(map[string]string)
		json.Unmarshal([]byte(interfaceServer.Config), &config)
		result.Config = config
	}
	return result
}

package interfaces

import (
	"context"

	"github.com/f-rambo/cloud-copilot/api/common"
	v1alpha1 "github.com/f-rambo/cloud-copilot/api/service/v1alpha1"
	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/utils"
)

type ServicesInterface struct {
	v1alpha1.UnimplementedServiceInterfaceServer
	serviceUc *biz.ServicesUseCase
}

func NewServicesInterface(serviceUc *biz.ServicesUseCase) *ServicesInterface {
	return &ServicesInterface{serviceUc: serviceUc}
}

func (s *ServicesInterface) List(ctx context.Context, serviceReq *v1alpha1.ServiceRequest) (*v1alpha1.Services, error) {
	if serviceReq.ProjectId == 0 {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	serviceData, total, err := s.serviceUc.List(ctx, serviceReq.ProjectId, serviceReq.Name, serviceReq.Page, serviceReq.PageSize)
	if err != nil {
		return nil, err
	}
	res := &v1alpha1.Services{}
	res.Total = total
	for _, v := range serviceData {
		serviceRes := &v1alpha1.Services{}
		err = utils.StructTransform(v, serviceRes)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (s *ServicesInterface) Save(ctx context.Context, service *v1alpha1.Service) (*common.Msg, error) {
	if service.ProjectId == 0 || service.Name == "" {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	bizService := &biz.Service{}
	err := utils.StructTransform(service, bizService)
	if err != nil {
		return nil, err
	}
	err = s.serviceUc.Save(ctx, bizService)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (s *ServicesInterface) Get(ctx context.Context, serviceReq *v1alpha1.ServiceRequest) (*v1alpha1.Service, error) {
	if serviceReq.Id == 0 {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	serviceData, err := s.serviceUc.Get(ctx, serviceReq.Id)
	if err != nil {
		return nil, err
	}
	serviceRes := &v1alpha1.Service{}
	err = utils.StructTransform(serviceData, serviceRes)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *ServicesInterface) Delete(ctx context.Context, serviceReq *v1alpha1.ServiceRequest) (*common.Msg, error) {
	if serviceReq.Id == 0 {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	err := s.serviceUc.Delete(ctx, serviceReq.Id)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (s *ServicesInterface) GetServiceResource(ctx context.Context, serviceReq *v1alpha1.ServiceRequest) (*v1alpha1.AlreadyResource, error) {
	if serviceReq.ProjectId == 0 {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	serviceResource, err := s.serviceUc.GetServiceResourceByProject(ctx, serviceReq.ProjectId)
	if err != nil {
		return nil, err
	}
	res := &v1alpha1.AlreadyResource{}
	err = utils.StructTransform(serviceResource, res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (s *ServicesInterface) SaveWorkflow(ctx context.Context, wf *v1alpha1.Workflow) (*common.Msg, error) {
	if wf.ServiceId == 0 || wf.Name == "" {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	bizWf := &biz.Workflow{}
	err := utils.StructTransform(wf, bizWf)
	if err != nil {
		return nil, err
	}
	err = s.serviceUc.SaveWorkflow(ctx, wf.ServiceId, bizWf)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (s *ServicesInterface) GetWorkflow(ctx context.Context, wfReq *v1alpha1.WorkflowRequest) (*v1alpha1.Workflow, error) {
	if wfReq.ServiceId == 0 || wfReq.WorkflowType == "" {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	wfType, ok := biz.WorkflowType_value[wfReq.WorkflowType]
	if !ok {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	wfData, err := s.serviceUc.GetWorkflow(ctx, wfReq.ServiceId, biz.WorkflowType(wfType))
	if err != nil {
		return nil, err
	}
	if wfData == nil || len(wfData.WorkflowSteps) == 0 {
		wfData, err = s.serviceUc.GetDefaultWorkflow(ctx, wfReq.ServiceId, biz.WorkflowType(wfType))
		if err != nil {
			return nil, err
		}
	}
	wfRes := &v1alpha1.Workflow{}
	err = utils.StructTransform(wfData, wfRes)
	if err != nil {
		return nil, err
	}
	return wfRes, nil
}

func (s *ServicesInterface) CreateContinuousIntegration(ctx context.Context, ci *v1alpha1.ContinuousIntegration) (*common.Msg, error) {
	if ci.ServiceId == 0 {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	bizCi := &biz.ContinuousIntegration{}
	err := utils.StructTransform(ci, bizCi)
	if err != nil {
		return nil, err
	}
	err = s.serviceUc.CreateContinuousIntegration(ctx, bizCi)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (s *ServicesInterface) GetContinuousIntegration(ctx context.Context, ciReq *v1alpha1.ContinuousIntegrationRequest) (*v1alpha1.ContinuousIntegration, error) {
	if ciReq.ServiceId == 0 || ciReq.Id == 0 {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	ciData, wf, err := s.serviceUc.GetContinuousIntegration(ctx, ciReq.Id)
	if err != nil {
		return nil, err
	}
	wfRes := &v1alpha1.Workflow{}
	err = utils.StructTransform(wf, wfRes)
	if err != nil {
		return nil, err
	}
	ciRes := &v1alpha1.ContinuousIntegration{}
	err = utils.StructTransform(ciData, ciRes)
	if err != nil {
		return nil, err
	}
	ciRes.Workflow = wfRes
	return ciRes, nil
}

func (s *ServicesInterface) GetContinuousIntegrations(ctx context.Context, ciReq *v1alpha1.ContinuousIntegrationRequest) (*v1alpha1.ContinuousIntegrations, error) {
	if ciReq.ServiceId == 0 {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	ciData, total, err := s.serviceUc.GetContinuousIntegrations(ctx, ciReq.ServiceId, ciReq.Page, ciReq.PageSize)
	if err != nil {
		return nil, err
	}
	ciRes := &v1alpha1.ContinuousIntegrations{}
	ciRes.Total = total
	for _, v := range ciData {
		ciVal := &v1alpha1.ContinuousIntegration{}
		err = utils.StructTransform(v, ciVal)
		if err != nil {
			return nil, err
		}
		ciRes.ContinuousIntegrations = append(ciRes.ContinuousIntegrations, ciVal)
	}
	return ciRes, nil
}

func (s *ServicesInterface) DeleteContinuousIntegration(ctx context.Context, ciReq *v1alpha1.ContinuousIntegrationRequest) (*common.Msg, error) {
	if ciReq.ServiceId == 0 || ciReq.Id == 0 {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	err := s.serviceUc.DeleteContinuousIntegration(ctx, ciReq.Id)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (s *ServicesInterface) CreateContinuousDeployment(ctx context.Context, cd *v1alpha1.ContinuousDeployment) (*common.Msg, error) {
	if cd.ServiceId == 0 || cd.Config == "" || cd.ConfigPath == "" {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	bizCd := &biz.ContinuousDeployment{}
	err := utils.StructTransform(cd, bizCd)
	if err != nil {
		return nil, err
	}
	err = s.serviceUc.CreateContinuousDeployment(ctx, bizCd)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (s *ServicesInterface) GetContinuousDeployment(ctx context.Context, cdReq *v1alpha1.ContinuousDeploymentRequest) (*v1alpha1.ContinuousDeployment, error) {
	if cdReq.ServiceId == 0 || cdReq.Id == 0 {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	cdData, wf, err := s.serviceUc.GetContinuousDeployment(ctx, cdReq.Id)
	if err != nil {
		return nil, err
	}
	wfRes := &v1alpha1.Workflow{}
	err = utils.StructTransform(wf, wfRes)
	if err != nil {
		return nil, err
	}
	cdRes := &v1alpha1.ContinuousDeployment{}
	err = utils.StructTransform(cdData, cdRes)
	if err != nil {
		return nil, err
	}
	cdRes.Workflow = wfRes
	return cdRes, nil
}

func (s *ServicesInterface) GetContinuousDeployments(ctx context.Context, cdReq *v1alpha1.ContinuousDeploymentRequest) (*v1alpha1.ContinuousDeployments, error) {
	if cdReq.ServiceId == 0 {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	cdData, total, err := s.serviceUc.GetContinuousDeployments(ctx, cdReq.ServiceId, cdReq.Page, cdReq.PageSize)
	if err != nil {
		return nil, err
	}
	cdRes := &v1alpha1.ContinuousDeployments{}
	cdRes.Total = total
	for _, v := range cdData {
		cdVal := &v1alpha1.ContinuousDeployment{}
		err = utils.StructTransform(v, cdVal)
		if err != nil {
			return nil, err
		}
		cdRes.ContinuousDeployments = append(cdRes.ContinuousDeployments, cdVal)
	}
	return cdRes, nil
}

func (s *ServicesInterface) DeleteContinuousDeployment(ctx context.Context, cdReq *v1alpha1.ContinuousDeploymentRequest) (*common.Msg, error) {
	if cdReq.ServiceId == 0 || cdReq.Id == 0 {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	err := s.serviceUc.DeleteContinuousDeployment(ctx, cdReq.Id)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (s *ServicesInterface) ApplyService(ctx context.Context, serviceReq *v1alpha1.ServiceRequest) (*common.Msg, error) {
	if serviceReq.Id == 0 || serviceReq.CiId == 0 || serviceReq.CdId == 0 {
		return nil, common.ResponseError(common.ErrorReason_ErrInvalidArgument)
	}
	err := s.serviceUc.ApplyService(ctx, serviceReq.Id, serviceReq.CiId, serviceReq.CdId)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

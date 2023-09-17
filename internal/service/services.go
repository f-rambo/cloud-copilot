package service

import (
	"context"
	"errors"

	v1 "github.com/f-rambo/ocean/api/service/v1"
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ServicesService struct {
	v1.UnimplementedServiceServiceServer
	uc *biz.ServicesUseCase
	c  *conf.Data
}

func NewServicesService(uc *biz.ServicesUseCase, c *conf.Data) *ServicesService {
	return &ServicesService{uc: uc, c: c}
}

func (s *ServicesService) SaveService(ctx context.Context, req *v1.Service) (*v1.ServiceID, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if req.Workflow == "" {
		return nil, errors.New("workflow is empty")
	}
	if req.Name == "" {
		return nil, errors.New("name is empty")
	}
	service := s.serviceTransformationBiz(req)
	err := s.uc.SaveService(ctx, service)
	if err != nil {
		return nil, err
	}
	return &v1.ServiceID{Id: int32(service.ID)}, nil
}

func (s *ServicesService) GetService(ctx context.Context, req *v1.ServiceID) (*v1.Service, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if req.Id == 0 {
		return nil, errors.New("id is empty")
	}
	service, err := s.uc.GetService(ctx, int(req.Id))
	if err != nil {
		return nil, err
	}
	data := s.serviceTransformationApi(service)
	data.Cis = make([]*v1.CI, 0)
	for _, v := range service.CIItems {
		data.Cis = append(data.Cis, s.ciTransformationApi(v))
	}
	return data, nil
}

func (s *ServicesService) GetServices(ctx context.Context, _ *emptypb.Empty) (*v1.Services, error) {
	data := make([]*v1.Service, 0)
	services, err := s.uc.GetServices(ctx)
	if err != nil {
		return nil, err
	}
	for _, v := range services {
		data = append(data, s.serviceTransformationApi(v))
	}
	return &v1.Services{Services: data}, nil
}

func (s *ServicesService) DeleteService(ctx context.Context, req *v1.ServiceID) (*v1.Msg, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if req.Id == 0 {
		return nil, errors.New("id is empty")
	}
	err := s.uc.DeleteService(ctx, int(req.Id))
	if err != nil {
		return nil, err
	}
	return &v1.Msg{Message: "success"}, nil
}

func (s *ServicesService) SaveCI(ctx context.Context, req *v1.CI) (*v1.CIID, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	if req.ServiceId == 0 {
		return nil, errors.New("service id is empty")
	}
	ci := s.ciTransformationBiz(req)
	err := s.uc.SaveCI(ctx, ci)
	if err != nil {
		return nil, err
	}
	return &v1.CIID{Id: int32(ci.ID)}, nil
}

func (s *ServicesService) GetCI(ctx context.Context, req *v1.CIID) (*v1.CI, error) {
	if req == nil || req.Id == 0 {
		return nil, errors.New("request is nil")
	}
	ci, err := s.uc.GetCI(ctx, int(req.Id))
	if err != nil {
		return nil, err
	}
	return s.ciTransformationApi(ci), nil
}

func (s *ServicesService) GetCIs(ctx context.Context, req *v1.ServiceID) (*v1.CIs, error) {
	if req == nil || req.Id == 0 {
		return nil, errors.New("request is nil")
	}
	cis, err := s.uc.GetCIs(ctx, int(req.Id))
	if err != nil {
		return nil, err
	}
	data := make([]*v1.CI, 0)
	for _, v := range cis {
		data = append(data, s.ciTransformationApi(v))
	}
	return &v1.CIs{CIs: data}, nil
}

func (s *ServicesService) DeleteCI(ctx context.Context, req *v1.CIID) (*v1.Msg, error) {
	if req == nil || req.Id == 0 {
		return nil, errors.New("request is nil")
	}
	err := s.uc.DeleteCI(ctx, int(req.Id))
	if err != nil {
		return nil, err
	}
	return &v1.Msg{Message: "success"}, nil
}

func (s *ServicesService) Deploy(ctx context.Context, req *v1.CIID) (*v1.Msg, error) {
	if req == nil || req.Id == 0 {
		return nil, errors.New("request is nil")
	}
	err := s.uc.Deploy(ctx, int(req.Id))
	if err != nil {
		return nil, err
	}
	return &v1.Msg{}, nil
}

func (s *ServicesService) GetOceanService(ctx context.Context, _ *emptypb.Empty) (*v1.Service, error) {
	service, err := s.uc.GetOceanService(ctx)
	if err != nil {
		return nil, err
	}
	data := s.serviceTransformationApi(service)
	for _, v := range service.CIItems {
		data.Cis = append(data.Cis, s.ciTransformationApi(v))
	}
	return data, nil
}

func (s *ServicesService) serviceTransformationBiz(service *v1.Service) *biz.Service {
	return &biz.Service{
		ID:           int(service.Id),
		Name:         service.Name,
		Repo:         service.Repo,
		Registry:     service.Registry,
		RegistryUser: service.RegistryUser,
		RegistryPwd:  service.RegistryPwd,
		CIItems:      make([]*biz.CI, 0),
		Workflow:     service.Workflow,
	}
}

func (s *ServicesService) serviceTransformationApi(service *biz.Service) *v1.Service {
	return &v1.Service{
		Id:           int32(service.ID),
		Name:         service.Name,
		Repo:         service.Repo,
		Registry:     service.Registry,
		RegistryUser: service.RegistryUser,
		RegistryPwd:  service.RegistryPwd,
		Workflow:     service.Workflow,
	}
}

func (s *ServicesService) ciTransformationBiz(ci *v1.CI) *biz.CI {
	return &biz.CI{
		ID:          int(ci.Id),
		Version:     ci.Version,
		Branch:      ci.Branch,
		Tag:         ci.Tag,
		Args:        ci.Args,
		Description: ci.Description,
		ServiceID:   int(ci.ServiceId),
	}
}

func (s *ServicesService) ciTransformationApi(ci *biz.CI) *v1.CI {
	return &v1.CI{
		Id:           int32(ci.ID),
		Version:      ci.Version,
		Branch:       ci.Branch,
		Tag:          ci.Tag,
		Args:         ci.Args,
		Description:  ci.Description,
		ServiceId:    int32(ci.ServiceID),
		WorkflowName: ci.WorkflowName,
	}
}

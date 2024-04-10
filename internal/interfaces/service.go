package interfaces

import (
	"context"
	"encoding/json"

	v1alpha1 "github.com/f-rambo/ocean/api/service/v1alpha1"
	"github.com/f-rambo/ocean/internal/biz"
)

type ServicesInterface struct {
	v1alpha1.UnimplementedServiceInterfaceServer
	uc *biz.ServicesUseCase
}

func NewServicesInterface(uc *biz.ServicesUseCase) *ServicesInterface {
	return &ServicesInterface{uc: uc}
}

func (s *ServicesInterface) List(ctx context.Context, serviceParam *v1alpha1.ServiceRequest) (*v1alpha1.Services, error) {
	services, itemCount, err := s.uc.List(ctx, &biz.Service{
		Name:      serviceParam.Name,
		ProjectID: int64(serviceParam.ProjectId),
	}, int(serviceParam.Page), int(serviceParam.PageSize))
	if err != nil {
		return nil, err
	}
	servicesInterfaces, err := s.bizToInterfaces(services)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.Services{
		Services: servicesInterfaces,
		Total:    int32(itemCount),
	}, nil
}

func (s *ServicesInterface) Save(ctx context.Context, serviceParam *v1alpha1.Service) (*v1alpha1.Msg, error) {
	serviceBiz, err := s.interfaceToBiz(serviceParam)
	if err != nil {
		return nil, err
	}
	err = s.uc.Save(ctx, serviceBiz)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.Msg{}, nil

}

func (s *ServicesInterface) bizToInterfaces(services []*biz.Service) ([]*v1alpha1.Service, error) {
	servicesInterface := make([]*v1alpha1.Service, 0)
	jsonServices, err := json.Marshal(services)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(jsonServices, &servicesInterface)
	if err != nil {
		return nil, err
	}
	return servicesInterface, nil
}

func (s *ServicesInterface) interfaceToBiz(service *v1alpha1.Service) (*biz.Service, error) {
	serviceBiz := &biz.Service{}
	jsonService, err := json.Marshal(service)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(jsonService, serviceBiz)
	if err != nil {
		return nil, err
	}
	return serviceBiz, nil
}

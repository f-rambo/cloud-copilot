package interfaces

import (
	"context"

	v1alpha1 "github.com/f-rambo/ocean/api/service/v1alpha1"
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/pkg/errors"
)

type ServicesInterface struct {
	v1alpha1.UnimplementedServiceInterfaceServer
	serviceUc *biz.ServicesUseCase
	projectUc *biz.ProjectUsecase
}

func NewServicesInterface(serviceUc *biz.ServicesUseCase, projectUc *biz.ProjectUsecase) *ServicesInterface {
	return &ServicesInterface{serviceUc: serviceUc, projectUc: projectUc}
}

func (s *ServicesInterface) List(ctx context.Context, serviceParam *v1alpha1.ServiceRequest) (*v1alpha1.Services, error) {
	services, itemCount, err := s.serviceUc.List(ctx, &biz.Service{
		Name:      serviceParam.Name,
		ProjectID: serviceParam.ProjectId,
	}, int(serviceParam.Page), int(serviceParam.PageSize))
	if err != nil {
		return nil, err
	}
	projectIds := make([]int64, 0)
	for _, v := range services {
		projectIds = append(projectIds, v.ProjectID)
	}
	projects, err := s.projectUc.ListByIds(ctx, projectIds)
	if err != nil {
		return nil, err
	}
	projectMap := make(map[int64]string)
	for _, v := range projects {
		projectMap[v.ID] = v.Name
	}
	interfaceServices := make([]*v1alpha1.Service, 0)
	for _, service := range services {
		interfaceServices = append(interfaceServices, s.bizToInterface(service))
	}
	for _, v := range interfaceServices {
		projectName, ok := projectMap[v.ProjectID]
		if ok {
			v.ProjectName = projectName
		}
	}
	return &v1alpha1.Services{
		Services: interfaceServices,
		Total:    itemCount,
	}, nil
}

func (s *ServicesInterface) Save(ctx context.Context, serviceParam *v1alpha1.Service) (*v1alpha1.Msg, error) {
	err := s.serviceUc.Save(ctx, s.interfaceToBiz(serviceParam))
	if err != nil {
		return nil, err
	}
	return &v1alpha1.Msg{}, nil
}

func (s *ServicesInterface) Get(ctx context.Context, serviceParam *v1alpha1.ServiceRequest) (*v1alpha1.Service, error) {
	service, err := s.serviceUc.Get(ctx, serviceParam.Id)
	if err != nil {
		return nil, err
	}
	return s.bizToInterface(service), nil
}

func (s *ServicesInterface) Delete(ctx context.Context, serviceParam *v1alpha1.ServiceRequest) (*v1alpha1.Msg, error) {
	if serviceParam.Id == 0 {
		return nil, errors.New("id is required")
	}

	err := s.serviceUc.Delete(ctx, serviceParam.Id)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.Msg{}, nil
}

func (s *ServicesInterface) bizToInterface(service *biz.Service) *v1alpha1.Service {
	servicesInterface := &v1alpha1.Service{
		ID:          service.ID,
		Name:        service.Name,
		CodeRepo:    service.CodeRepo,
		Replicas:    service.Replicas,
		CPU:         service.CPU,
		LimitCpu:    service.LimitCpu,
		GPU:         service.GPU,
		LimitGPU:    service.LimitGPU,
		Memory:      service.Memory,
		LimitMemory: service.LimitMemory,
		Disk:        service.Disk,
		LimitDisk:   service.LimitDisk,
		WorkflowID:  service.WorkflowID,
		ProjectID:   service.ProjectID,
		Business:    service.Business,
		Technology:  service.Technology,
		Ports:       make([]*v1alpha1.Port, 0),
	}
	for _, port := range service.Ports {
		port := &v1alpha1.Port{
			ID:            port.ID,
			IngressPath:   port.IngressPath,
			Protocol:      port.Protocol,
			ContainerPort: port.ContainerPort,
		}
		servicesInterface.Ports = append(servicesInterface.Ports, port)
	}
	return servicesInterface
}

func (s *ServicesInterface) interfaceToBiz(service *v1alpha1.Service) *biz.Service {
	ports := make([]biz.Port, 0)
	for _, port := range service.Ports {
		ports = append(ports, biz.Port{
			ID:            port.ID,
			IngressPath:   port.IngressPath,
			Protocol:      port.Protocol,
			ContainerPort: port.ContainerPort,
		})
	}
	return &biz.Service{
		ID:          service.ID,
		Name:        service.Name,
		CodeRepo:    service.CodeRepo,
		Replicas:    service.Replicas,
		CPU:         service.CPU,
		LimitCpu:    service.LimitCpu,
		GPU:         service.GPU,
		LimitGPU:    service.LimitGPU,
		Memory:      service.Memory,
		LimitMemory: service.LimitMemory,
		Disk:        service.Disk,
		LimitDisk:   service.LimitDisk,
		WorkflowID:  service.WorkflowID,
		ProjectID:   service.ProjectID,
		Business:    service.Business,
		Technology:  service.Technology,
		Ports:       ports,
	}
}

package service

import (
	"context"

	v1 "github.com/f-rambo/ocean/api/service/v1"
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
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

func (s *ServicesService) GetOceanService(ctx context.Context, _ *emptypb.Empty) (*v1.Service, error) {
	service, err := s.uc.GetOceanService(ctx)
	if err != nil {
		return nil, err
	}
	wf, err := utils.GetDefaultWorkflowStr()
	if err != nil {
		return nil, err
	}
	data := &v1.Service{
		Id:           int32(service.ID),
		Name:         service.Name,
		Repo:         service.Repo,
		Registry:     service.Registry,
		RegistryUser: service.RegistryUser,
		RegistryPwd:  service.RegistryPwd,
		Workflow:     wf,
	}
	for _, v := range service.CIItems {
		data.Cis = append(data.Cis, &v1.CI{
			Id:          int32(v.ID),
			Version:     v.Version,
			Branch:      v.Branch,
			Tag:         v.Tag,
			Args:        v.Args,
			Description: v.Description,
			ServiceId:   int32(v.ServiceID),
		})
		break
	}
	return data, nil
}

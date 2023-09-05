package service

import (
	"context"
	v1 "ocean/api/infra/v1"
	"ocean/internal/biz"

	"github.com/golang/protobuf/ptypes/empty"
)

type InfraService struct {
	//v1.UnimplementedInfraServer
	v1.UnimplementedInfraServer
	//uc *biz.InfraUsecase
	uc *biz.InfraUsecase
}

func NewInfraService(uc *biz.InfraUsecase) *InfraService {
	return &InfraService{uc: uc}
}

func (s *InfraService) GetConfig(ctx context.Context, _ *empty.Empty) (*v1.InfraRes, error) {
	data, err := s.uc.GetInfra(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.InfraRes{
		ScriptPath:          data.ScriptPath,
		KubesprayVersion:    data.KubesprayVersion,
		KubesprayPath:       data.KubesprayPath,
		KubesparyPackageTag: data.KubesprayPkgTag,
	}, nil
}

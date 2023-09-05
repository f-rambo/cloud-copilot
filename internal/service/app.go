package service

import (
	"context"
	v1 "ocean/api/app/v1"
	"ocean/internal/biz"

	"github.com/golang/protobuf/ptypes/empty"
)

type AppService struct {
	v1.UnimplementedAppServiceServer
	uc *biz.AppUsecase
}

func NewAppService(uc *biz.AppUsecase) *AppService {
	return &AppService{uc: uc}
}

func (a *AppService) GetApps(ctx context.Context, _ *empty.Empty) (*v1.App, error) {
	_, err := a.uc.GetApps(ctx)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (a *AppService) SaveApps(ctx context.Context, apps *v1.App) (*v1.Msg, error) {

	return &v1.Msg{Message: "success"}, nil
}

func (a *AppService) GetAppConfig(ctx context.Context, req *v1.GetAppConfigRequest) (*v1.GetAppConfigResponse, error) {
	_, err := a.uc.GetAppConfig(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.GetAppConfigResponse{}, nil
}

func (a *AppService) SaveAppConfig(ctx context.Context, req *v1.SaveAppConfigRequest) (*v1.Msg, error) {

	return &v1.Msg{Message: "success"}, nil
}

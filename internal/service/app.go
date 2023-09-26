package service

import (
	"context"
	"errors"

	v1alpha1 "github.com/f-rambo/ocean/api/app/v1alpha1"
	"github.com/f-rambo/ocean/internal/biz"
)

type AppService struct {
	v1alpha1.UnimplementedAppServiceServer
	uc *biz.AppUsecase
}

func NewAppService(uc *biz.AppUsecase) *AppService {
	return &AppService{uc: uc}
}

func (a *AppService) GetApps(ctx context.Context, clusterID *v1alpha1.ClusterID) (*v1alpha1.Apps, error) {
	if clusterID == nil || clusterID.ClusterID == 0 {
		return nil, errors.New("cluster id is require")
	}
	apps, err := a.uc.GetApps(ctx, int(clusterID.ClusterID))
	if err != nil {
		return nil, err
	}
	data := make([]*v1alpha1.App, 0)
	for _, app := range apps {
		data = append(data, &v1alpha1.App{
			Id:        int32(app.ID),
			Name:      app.Name,
			RepoName:  app.RepoName,
			RepoUrl:   app.RepoURL,
			ChartName: app.ChartName,
			Version:   app.Version,
			Namespace: app.Namespace,
			ClusterID: int32(app.ClusterID),
		})
	}
	return &v1alpha1.Apps{Apps: data}, nil
}

func (a *AppService) GetApp(ctx context.Context, appID *v1alpha1.AppID) (*v1alpha1.App, error) {
	if appID == nil || appID.AppID == 0 {
		return nil, errors.New("app id is require")
	}
	app, err := a.uc.GetApp(ctx, int(appID.AppID))
	if err != nil {
		return nil, err
	}
	return &v1alpha1.App{
		Id:        int32(app.ID),
		Name:      app.Name,
		RepoName:  app.RepoName,
		RepoUrl:   app.RepoURL,
		ChartName: app.ChartName,
		Version:   app.Version,
		Namespace: app.Namespace,
		ClusterID: int32(app.ClusterID),
		Config:    app.Config,
		Secret:    app.Secret,
	}, nil
}

func (a *AppService) Save(ctx context.Context, app *v1alpha1.App) (*v1alpha1.AppID, error) {
	if app == nil {
		return nil, errors.New("app is require")
	}
	if app.ClusterID == 0 {
		return nil, errors.New("cluster id is require")
	}
	bizApp := &biz.App{
		ID:        int(app.Id),
		Name:      app.Name,
		RepoName:  app.RepoName,
		RepoURL:   app.RepoUrl,
		ChartName: app.ChartName,
		Version:   app.Version,
		Namespace: app.Namespace,
		ClusterID: int(app.ClusterID),
		Config:    app.Config,
		Secret:    app.Secret,
	}
	err := a.uc.Save(ctx, bizApp)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.AppID{AppID: int32(bizApp.ID), ClusterID: int32(bizApp.ClusterID)}, nil
}

func (a *AppService) Apply(ctx context.Context, appID *v1alpha1.AppID) (*v1alpha1.Msg, error) {
	if appID == nil || appID.AppID == 0 {
		return nil, errors.New("app id is require")
	}
	err := a.uc.Apply(ctx, int(appID.AppID))
	if err != nil {
		return nil, err
	}
	return &v1alpha1.Msg{Message: "ok"}, nil
}

func (a *AppService) Delete(ctx context.Context, appID *v1alpha1.AppID) (*v1alpha1.Msg, error) {
	if appID == nil || appID.AppID == 0 {
		return nil, errors.New("app id is require")
	}
	err := a.uc.DeleteApp(ctx, int(appID.AppID))
	if err != nil {
		return nil, err
	}
	return &v1alpha1.Msg{Message: "ok"}, nil
}

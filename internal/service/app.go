package service

import (
	"context"
	"encoding/json"
	"errors"

	v1 "github.com/f-rambo/ocean/api/app/v1"
	"github.com/f-rambo/ocean/internal/biz"
	"gopkg.in/yaml.v3"
	coreV1 "k8s.io/api/core/v1"
)

type AppService struct {
	v1.UnimplementedAppServiceServer
	uc *biz.AppUsecase
}

func NewAppService(uc *biz.AppUsecase) *AppService {
	return &AppService{uc: uc}
}

func (a *AppService) GetApps(ctx context.Context, clusterID *v1.ClusterID) (*v1.Apps, error) {
	if clusterID == nil || clusterID.ClusterID == 0 {
		return nil, errors.New("cluster id is require")
	}
	apps, err := a.uc.GetApps(ctx, int(clusterID.ClusterID))
	if err != nil {
		return nil, err
	}
	data := make([]*v1.App, 0)
	for _, app := range apps {
		data = append(data, &v1.App{
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
	return &v1.Apps{Apps: data}, nil
}

func (a *AppService) GetApp(ctx context.Context, appID *v1.AppID) (*v1.App, error) {
	if appID == nil || appID.AppID == 0 {
		return nil, errors.New("app id is require")
	}
	app, err := a.uc.GetApp(ctx, int(appID.AppID))
	if err != nil {
		return nil, err
	}
	var configStr []byte
	if app.ConfigMap != nil && len(app.ConfigMap.Data) > 0 {
		configStr, err = json.Marshal(app.ConfigMap.Data)
		if err != nil {
			return nil, err
		}
	}
	var secretStr []byte
	if app.Secret != nil && len(app.Secret.Data) > 0 {
		secretStr, err = json.Marshal(app.Secret.Data)
		if err != nil {
			return nil, err
		}
	}
	return &v1.App{
		Id:        int32(app.ID),
		Name:      app.Name,
		RepoName:  app.RepoName,
		RepoUrl:   app.RepoURL,
		ChartName: app.ChartName,
		Version:   app.Version,
		Namespace: app.Namespace,
		ClusterID: int32(app.ClusterID),
		Config:    string(configStr),
		Secret:    string(secretStr),
	}, nil
}

func (a *AppService) Save(ctx context.Context, app *v1.App) (*v1.Msg, error) {
	if app == nil {
		return nil, errors.New("app is require")
	}
	if app.ClusterID == 0 {
		return nil, errors.New("cluster id is require")
	}
	var configMap *coreV1.ConfigMap
	if app.Config != "" {
		configMap.Data = map[string]string{
			"config": app.Config, //yaml
		}
	}
	var secret *coreV1.Secret
	if app.Secret != "" {
		secretData := make(map[string][]byte)
		err := yaml.Unmarshal([]byte(app.Secret), &secretData)
		if err != nil {
			return nil, err
		}
		secret.Data = secretData
	}
	err := a.uc.Save(ctx, &biz.App{
		Name:      app.Name,
		RepoName:  app.RepoName,
		RepoURL:   app.RepoUrl,
		ChartName: app.ChartName,
		Version:   app.Version,
		Namespace: app.Namespace,
		ClusterID: int(app.ClusterID),
		ConfigMap: configMap,
		Secret:    secret,
	})
	if err != nil {
		return nil, err
	}
	return &v1.Msg{Message: "ok"}, nil
}

func (a *AppService) Apply(ctx context.Context, appID *v1.AppID) (*v1.Msg, error) {
	if appID == nil || appID.AppID == 0 {
		return nil, errors.New("app id is require")
	}
	err := a.uc.Apply(ctx, int(appID.AppID))
	if err != nil {
		return nil, err
	}
	return &v1.Msg{Message: "ok"}, nil
}

func (a *AppService) Delete(ctx context.Context, appID *v1.AppID) (*v1.Msg, error) {
	if appID == nil || appID.AppID == 0 {
		return nil, errors.New("app id is require")
	}
	err := a.uc.DeleteApp(ctx, int(appID.AppID))
	if err != nil {
		return nil, err
	}
	return &v1.Msg{Message: "ok"}, nil
}

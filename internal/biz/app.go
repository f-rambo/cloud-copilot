package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	coreV1 "k8s.io/api/core/v1"
)

type App struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	RepoName  string           `json:"repoName,omitempty"`
	RepoURL   string           `json:"repoURL,omitempty"`
	ChartName string           `json:"chartName,omitempty"`
	Version   string           `json:"version,omitempty"`
	ConfigMap coreV1.ConfigMap `json:"configMapName,omitempty"`
	Secret    coreV1.Secret    `json:"secretName,omitempty"`
	Namespace string           `json:"namespace,omitempty"`
}

type AppRepo interface {
	GetApps(context.Context) ([]*App, error)
	SaveApp(context.Context, *App) error
	DeployApp(context.Context, string) error
	DestroyApp(context.Context, string) error
	GetAppById(context.Context, string) (*App, error)
}

type AppUsecase struct {
	repo AppRepo
	log  *log.Helper
}

func NewAppUsecase(repo AppRepo, logger log.Logger) *AppUsecase {
	return &AppUsecase{repo: repo, log: log.NewHelper(logger)}
}

// 保存APP配置文件
func (a *AppUsecase) SaveApp(ctx context.Context, apps *App) error {
	return a.repo.SaveApp(ctx, apps)
}

// 获取APP配置文件
func (a *AppUsecase) GetApps(ctx context.Context) ([]*App, error) {
	return a.repo.GetApps(ctx)
}

func (a *AppUsecase) GetAppConfig(ctx context.Context, id string) (map[string]string, error) {
	app, err := a.repo.GetAppById(ctx, id)
	if err != nil {
		return nil, err
	}
	return app.ConfigMap.Data, nil
}

func (a *AppUsecase) SaveAppConfig(ctx context.Context, id string, config map[string]string) error {
	app, err := a.repo.GetAppById(ctx, id)
	if err != nil {
		return err
	}
	app.ConfigMap.Data = config
	return a.repo.SaveApp(ctx, app)
}

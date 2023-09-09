package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	coreV1 "k8s.io/api/core/v1"
)

type App struct {
	ID        int              `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name      string           `json:"name" gorm:"column:name; default:''; NOT NULL"`
	RepoName  string           `json:"repoName,omitempty" gorm:"column:repo_name; default:''; NOT NULL"`
	RepoURL   string           `json:"repoURL,omitempty" gorm:"column:repo_url; default:''; NOT NULL"`
	ChartName string           `json:"chartName,omitempty" gorm:"column:chart_name; default:''; NOT NULL"`
	Version   string           `json:"version,omitempty" gorm:"column:version; default:''; NOT NULL"`
	ConfigMap coreV1.ConfigMap `json:"configMap,omitempty" gorm:"column:config_map; type:json"`
	Secret    coreV1.Secret    `json:"secret,omitempty" gorm:"column:secret; type:json"`
	Namespace string           `json:"namespace,omitempty" gorm:"column:namespace; default:''; NOT NULL"`
	gorm.Model
}

type AppRepo interface {
	GetApps(context.Context) ([]*App, error)
	SaveApp(context.Context, *App) error
	DeployApp(context.Context, string) error
	DestroyApp(context.Context, string) error
	GetAppById(context.Context, int) (*App, error)
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
	// app, err := a.repo.GetAppById(ctx, id)
	// if err != nil {
	// 	return nil, err
	// }
	// return app.ConfigMap.Data, nil
	return nil, nil
}

func (a *AppUsecase) SaveAppConfig(ctx context.Context, id string, config map[string]string) error {
	// app, err := a.repo.GetAppById(ctx, id)
	// if err != nil {
	// 	return err
	// }
	// app.ConfigMap.Data = config
	// return a.repo.SaveApp(ctx, app)
	return nil
}

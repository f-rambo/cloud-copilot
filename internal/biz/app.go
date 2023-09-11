package biz

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
	coreV1 "k8s.io/api/core/v1"
)

type App struct {
	ID        int               `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name      string            `json:"name" gorm:"column:name; default:''; NOT NULL"`
	RepoName  string            `json:"repoName,omitempty" gorm:"column:repo_name; default:''; NOT NULL"`
	RepoURL   string            `json:"repoURL,omitempty" gorm:"column:repo_url; default:''; NOT NULL"`
	ChartName string            `json:"chartName,omitempty" gorm:"column:chart_name; default:''; NOT NULL"`
	Version   string            `json:"version,omitempty" gorm:"column:version; default:''; NOT NULL"`
	ConfigMap *coreV1.ConfigMap `json:"configMap,omitempty" gorm:"column:config_map; type:json"`
	Secret    *coreV1.Secret    `json:"secret,omitempty" gorm:"column:secret; type:json"`
	Namespace string            `json:"namespace,omitempty" gorm:"column:namespace; default:''; NOT NULL"`
	Deployed  bool              `json:"deployed,omitempty" gorm:"column:deployed; default:false; NOT NULL"`
	ClusterID int               `json:"cluster_id,omitempty" gorm:"column:cluster_id; default:0; NOT NULL"`
	gorm.Model
}

type AppRepo interface {
	Save(context.Context, *App) error
	GetApp(context.Context, int) (*App, error)
	GetApps(context.Context, int) ([]*App, error)
	DeleteApp(context.Context, int) error
	Apply(context.Context, *App) error
}

type AppUsecase struct {
	repo AppRepo
	log  *log.Helper
}

func NewAppUsecase(repo AppRepo, logger log.Logger) *AppUsecase {
	return &AppUsecase{repo: repo, log: log.NewHelper(logger)}
}

func (a *AppUsecase) Save(ctx context.Context, app *App) error {
	return a.repo.Save(ctx, app)
}

func (a *AppUsecase) GetApps(ctx context.Context) ([]*App, error) {
	return a.repo.GetApps(ctx, 0)
}

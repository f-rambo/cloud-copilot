package clusterruntime

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
)

const (
	ServiceNameClusterRuntime = "cluster-runtime"
)

type ClusterRuntimeApp struct {
	conf *conf.Bootstrap
	log  *log.Helper
}

func NewClusterRuntimeApp(conf *conf.Bootstrap, logger log.Logger) biz.AppRuntime {
	return &ClusterRuntimeApp{
		conf: conf,
		log:  log.NewHelper(logger),
	}
}

func (c *ClusterRuntimeApp) CheckCluster(ctx context.Context) bool {
	return false
}

func (c *ClusterRuntimeApp) Init(ctx context.Context) ([]*biz.App, []*biz.AppRelease, error) {
	return nil, nil, nil
}

func (c *ClusterRuntimeApp) GetClusterResources(ctx context.Context, appRelease *biz.AppRelease) ([]*biz.AppReleaseResource, error) {
	return nil, nil
}

func (c *ClusterRuntimeApp) DeleteApp(ctx context.Context, app *biz.App) error {
	return nil
}

func (c *ClusterRuntimeApp) DeleteAppVersion(ctx context.Context, app *biz.App, appVersion *biz.AppVersion) error {
	return nil
}

func (c *ClusterRuntimeApp) GetAppAndVersionInfo(ctx context.Context, app *biz.App, appVersion *biz.AppVersion) error {
	return nil
}

func (c *ClusterRuntimeApp) AppRelease(ctx context.Context, app *biz.App, appVersion *biz.AppVersion, appRelease *biz.AppRelease, appRepo *biz.AppRepo) error {
	return nil
}

func (c *ClusterRuntimeApp) DeleteAppRelease(ctx context.Context, appRelease *biz.AppRelease) error {
	return nil
}

func (c *ClusterRuntimeApp) AddAppRepo(ctx context.Context, appRepo *biz.AppRepo) error {
	return nil
}

func (c *ClusterRuntimeApp) GetAppsByRepo(ctx context.Context, appRepo *biz.AppRepo) ([]*biz.App, error) {
	return nil, nil
}

func (c *ClusterRuntimeApp) GetAppDetailByRepo(ctx context.Context, apprepo *biz.AppRepo, appName, version string) (*biz.App, error) {
	return nil, nil
}

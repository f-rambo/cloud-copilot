package clusterruntime

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	appApi "github.com/f-rambo/cloud-copilot/internal/repository/clusterruntime/api/app"
	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/emptypb"
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
	grpconn, err := connGrpc(ctx, c.conf)
	if err != nil {
		return false
	}
	defer grpconn.Close()
	res, err := appApi.NewAppInterfaceClient(grpconn.Conn).CheckCluster(ctx, &emptypb.Empty{})
	if err != nil {
		return false
	}
	if res.Ok {
		return true
	}
	return false
}

func (c *ClusterRuntimeApp) DeleteApp(ctx context.Context, app *biz.App) error {
	grpconn, err := connGrpc(ctx, c.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	_, err = appApi.NewAppInterfaceClient(grpconn.Conn).DeleteApp(ctx, app)
	if err != nil {
		return err
	}
	return nil
}

func (c *ClusterRuntimeApp) DeleteAppVersion(ctx context.Context, app *biz.App, appVersion *biz.AppVersion) error {
	grpconn, err := connGrpc(ctx, c.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	_, err = appApi.NewAppInterfaceClient(grpconn.Conn).DeleteAppVersion(ctx, &appApi.DeleteAppVersionReq{
		App:     app,
		Version: appVersion,
	})
	if err != nil {
		return err
	}
	return nil
}

func (c *ClusterRuntimeApp) GetAppAndVersionInfo(ctx context.Context, app *biz.App, appVersion *biz.AppVersion) error {
	grpconn, err := connGrpc(ctx, c.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	res, err := appApi.NewAppInterfaceClient(grpconn.Conn).GetAppAndVersionInfo(ctx, &appApi.GetAppAndVersionInfo{
		App:     app,
		Version: appVersion,
	})
	if err != nil {
		return err
	}
	app.Name = res.App.Name
	app.Icon = res.App.Icon
	app.AppTypeId = res.App.AppTypeId
	appVersion.Name = res.Version.Name
	appVersion.Chart = res.Version.Chart
	appVersion.Version = res.Version.Version
	appVersion.DefaultConfig = res.Version.DefaultConfig
	return nil
}

func (c *ClusterRuntimeApp) AppRelease(ctx context.Context, app *biz.App, appVersion *biz.AppVersion, appRelease *biz.AppRelease, appRepo *biz.AppRepo) error {
	grpconn, err := connGrpc(ctx, c.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	res, err := appApi.NewAppInterfaceClient(grpconn.Conn).AppRelease(ctx, &appApi.AppReleaseReq{
		App:     app,
		Version: appVersion,
		Release: appRelease,
		Repo:    appRepo,
	})
	if err != nil {
		return err
	}
	appRelease.Id = res.Id
	appRelease.AppId = res.AppId
	appRelease.VersionId = res.VersionId
	appRelease.Status = res.Status
	return nil
}

func (c *ClusterRuntimeApp) DeleteAppRelease(ctx context.Context, appRelease *biz.AppRelease) error {
	grpconn, err := connGrpc(ctx, c.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	_, err = appApi.NewAppInterfaceClient(grpconn.Conn).DeleteAppRelease(ctx, appRelease)
	if err != nil {
		return err
	}
	return nil
}

func (c *ClusterRuntimeApp) AddAppRepo(ctx context.Context, appRepo *biz.AppRepo) error {
	grpconn, err := connGrpc(ctx, c.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	res, err := appApi.NewAppInterfaceClient(grpconn.Conn).AddAppRepo(ctx, appRepo)
	if err != nil {
		return err
	}
	appRepo.Id = res.Id
	appRepo.Name = res.Name
	appRepo.Url = res.Url
	return nil
}

func (c *ClusterRuntimeApp) GetAppsByRepo(ctx context.Context, appRepo *biz.AppRepo) ([]*biz.App, error) {
	grpconn, err := connGrpc(ctx, c.conf)
	if err != nil {
		return nil, err
	}
	defer grpconn.Close()
	res, err := appApi.NewAppInterfaceClient(grpconn.Conn).GetAppsByRepo(ctx, appRepo)
	if err != nil {
		return nil, err
	}
	return res.Apps, nil
}

func (c *ClusterRuntimeApp) GetAppDetailByRepo(ctx context.Context, apprepo *biz.AppRepo, appName, version string) (*biz.App, error) {
	grpconn, err := connGrpc(ctx, c.conf)
	if err != nil {
		return nil, err
	}
	defer grpconn.Close()
	res, err := appApi.NewAppInterfaceClient(grpconn.Conn).GetAppDetailByRepo(ctx, &appApi.GetAppDetailByRepoReq{
		Repo:    apprepo,
		AppName: appName,
		Version: version,
	})
	if err != nil {
		return nil, err
	}
	if res != nil {
		return res, nil
	}
	return nil, nil
}

func (c *ClusterRuntimeApp) ReloadAppReleaseResource(ctx context.Context, appReleaseResource *biz.AppReleaseResource) error {
	grpconn, err := connGrpc(ctx, c.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	_, err = appApi.NewAppInterfaceClient(grpconn.Conn).ReloadAppReleaseResource(ctx, appReleaseResource)
	if err != nil {
		return err
	}
	return nil
}

func (c *ClusterRuntimeApp) GetAppReleaseResources(ctx context.Context, appRelease *biz.AppRelease) error {
	grpconn, err := connGrpc(ctx, c.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	res, err := appApi.NewAppInterfaceClient(grpconn.Conn).GetAppReleaseResources(ctx, appRelease)
	if err != nil {
		return err
	}
	appRelease.Resources = res.Resources
	return nil
}

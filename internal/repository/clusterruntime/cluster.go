package clusterruntime

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	appApi "github.com/f-rambo/cloud-copilot/internal/repository/clusterruntime/api/app"
	clusterApi "github.com/f-rambo/cloud-copilot/internal/repository/clusterruntime/api/cluster"
	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/go-kratos/kratos/v2/log"
)

type ClusterRuntimeCluster struct {
	conf *conf.Bootstrap
	log  *log.Helper
}

func NewClusterRuntimeCluster(conf *conf.Bootstrap, logger log.Logger) biz.ClusterRuntime {
	return &ClusterRuntimeCluster{
		conf: conf,
		log:  log.NewHelper(logger),
	}
}

func (c *ClusterRuntimeCluster) CurrentCluster(ctx context.Context, cluster *biz.Cluster) error {
	grpconn, err := connGrpc(ctx, c.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	clusterInterfaceClient := clusterApi.NewClusterInterfaceClient(grpconn.Conn)
	res, err := clusterInterfaceClient.CheckClusterInstalled(ctx, cluster)
	if err != nil {
		return err
	}
	if !res.Installed {
		return biz.ErrClusterNotFound
	}
	clusterRes, err := clusterInterfaceClient.CurrentCluster(ctx, cluster)
	if err != nil {
		return err
	}
	err = utils.StructTransform(clusterRes, cluster)
	if err != nil {
		return err
	}
	return nil
}

func (c *ClusterRuntimeCluster) HandlerNodes(ctx context.Context, cluster *biz.Cluster) error {
	grpconn, err := connGrpc(ctx, c.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	clusterRes, err := clusterApi.NewClusterInterfaceClient(grpconn.Conn).HandlerNodes(ctx, cluster)
	if err != nil {
		return err
	}
	err = utils.StructTransform(clusterRes, cluster)
	if err != nil {
		return err
	}
	return nil
}

func (c *ClusterRuntimeCluster) InstallBasicComponent(ctx context.Context, basicComponent biz.BasicComponentAppType) ([]*biz.App, []*biz.AppRelease, error) {
	grpconn, err := connGrpc(ctx, c.conf)
	if err != nil {
		return nil, nil, err
	}
	defer grpconn.Close()
	res, err := appApi.NewAppInterfaceClient(grpconn.Conn).InstallBasicComponent(ctx, &appApi.InstallBasicComponentReq{
		BasicComponentAppType: basicComponent,
	})
	if err != nil {
		return nil, nil, err
	}
	return res.Apps, res.Releases, nil
}

func (c *ClusterRuntimeCluster) AppRelease(ctx context.Context, app *biz.App, version *biz.AppVersion, appRelease *biz.AppRelease) (*biz.AppRelease, error) {
	grpconn, err := connGrpc(ctx, c.conf)
	if err != nil {
		return nil, err
	}
	defer grpconn.Close()
	return appApi.NewAppInterfaceClient(grpconn.Conn).AppRelease(ctx, &appApi.AppReleaseReq{App: app, Version: version, Release: appRelease})
}

func (c *ClusterRuntimeCluster) ReloadAppReleaseResource(ctx context.Context, appReleaseResource *biz.AppReleaseResource) error {
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

func (c *ClusterRuntimeCluster) GetAppReleaseResources(ctx context.Context, appRelease *biz.AppRelease) error {
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

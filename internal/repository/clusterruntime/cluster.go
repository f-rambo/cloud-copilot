package clusterruntime

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
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

func (c *ClusterRuntimeCluster) MigrateToCluster(ctx context.Context, cluster *biz.Cluster) error {
	// Todo data migration
	grpconn, err := connGrpc(ctx, c.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	clusterRes, err := clusterApi.NewClusterInterfaceClient(grpconn.Conn).MigrateToCluster(ctx, cluster)
	if err != nil {
		return err
	}
	err = utils.StructTransform(clusterRes, cluster)
	if err != nil {
		return err
	}
	return nil
}

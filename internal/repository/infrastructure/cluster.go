package infrastructure

import (
	"context"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	clusterApi "github.com/f-rambo/cloud-copilot/internal/repository/infrastructure/api/cluster"
	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/go-kratos/kratos/v2/log"
)

const (
	ServiceNameInfrastructure = "infrastructure"
)

type InfrastructureCluster struct {
	conf *conf.Bootstrap
	log  *log.Helper
}

func NewInfrastructureCluster(conf *conf.Bootstrap, logger log.Logger) biz.ClusterInfrastructure {
	return &InfrastructureCluster{
		conf: conf,
		log:  log.NewHelper(logger),
	}
}

func (i *InfrastructureCluster) GetServiceConfig() *conf.Service {
	for _, service := range i.conf.Services {
		if service.Name == ServiceNameInfrastructure {
			return service
		}
	}
	return nil
}

func (i *InfrastructureCluster) GetRegions(ctx context.Context, cluster *biz.Cluster) error {
	service := i.GetServiceConfig()
	grpconn, err := new(utils.GrpcConn).OpenGrpcConn(ctx, service.Addr, service.Port)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	clusterRes, err := clusterApi.NewClusterInterfaceClient(grpconn.Conn).GetRegions(ctx, cluster)
	if err != nil {
		return err
	}
	cluster.CloudResources = clusterRes.CloudResources
	return nil
}

func (i *InfrastructureCluster) Start(ctx context.Context, cluster *biz.Cluster) error {
	return nil
}

func (i *InfrastructureCluster) Stop(ctx context.Context, cluster *biz.Cluster) error {
	return nil
}

func (i *InfrastructureCluster) MigrateToBostionHost(ctx context.Context, cluster *biz.Cluster) error {
	return nil
}

func (i *InfrastructureCluster) GetNodesSystemInfo(ctx context.Context, cluster *biz.Cluster) error {
	return nil
}

func (i *InfrastructureCluster) Install(ctx context.Context, cluster *biz.Cluster) error {
	return nil
}

func (i *InfrastructureCluster) UnInstall(ctx context.Context, cluster *biz.Cluster) error {
	return nil
}

func (i *InfrastructureCluster) HandlerNodes(ctx context.Context, cluster *biz.Cluster) error {
	return nil
}

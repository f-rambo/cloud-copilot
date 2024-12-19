package infrastructure

import (
	"context"
	"io"

	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	clusterApi "github.com/f-rambo/cloud-copilot/internal/repository/infrastructure/api/cluster"
	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/proto"
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

func (i *InfrastructureCluster) GetRegions(ctx context.Context, cluster *biz.Cluster) error {
	grpconn, err := coonGrpc(ctx, i.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	res, err := clusterApi.NewClusterInterfaceClient(grpconn.Conn).GetRegions(ctx, cluster)
	if err != nil {
		return err
	}
	cluster.DeleteCloudResource(biz.ResourceType_REGION)
	for _, v := range res.Resources {
		cluster.AddCloudResource(v)
	}
	return nil
}

func (i *InfrastructureCluster) GetZones(ctx context.Context, cluster *biz.Cluster) error {
	grpconn, err := coonGrpc(ctx, i.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	res, err := clusterApi.NewClusterInterfaceClient(grpconn.Conn).GetZones(ctx, cluster)
	if err != nil {
		return err
	}
	cluster.DeleteCloudResource(biz.ResourceType_AVAILABILITY_ZONES)
	for _, v := range res.Resources {
		cluster.AddCloudResource(v)
	}
	return nil
}

func (i *InfrastructureCluster) CreateCloudBasicResource(ctx context.Context, cluster *biz.Cluster) error {
	grpconn, err := coonGrpc(ctx, i.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	stream, err := clusterApi.NewClusterInterfaceClient(grpconn.Conn).CreateCloudBasicResource(ctx, cluster)
	if err != nil {
		return err
	}
	for {
		data, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		protoBuf, err := proto.Marshal(data)
		if err != nil {
			return err
		}
		err = proto.Unmarshal(protoBuf, cluster)
		if err != nil {
			return err
		}
	}
}

func (i *InfrastructureCluster) DeleteCloudBasicResource(ctx context.Context, cluster *biz.Cluster) error {
	grpconn, err := coonGrpc(ctx, i.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	stream, err := clusterApi.NewClusterInterfaceClient(grpconn.Conn).DeleteCloudBasicResource(ctx, cluster)
	if err != nil {
		return err
	}
	for {
		data, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		protoBuf, err := proto.Marshal(data)
		if err != nil {
			return err
		}
		err = proto.Unmarshal(protoBuf, cluster)
		if err != nil {
			return err
		}
	}
}

func (i *InfrastructureCluster) ManageNodeResource(ctx context.Context, cluster *biz.Cluster) error {
	grpconn, err := coonGrpc(ctx, i.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	stream, err := clusterApi.NewClusterInterfaceClient(grpconn.Conn).ManageNodeResource(ctx, cluster)
	if err != nil {
		return err
	}
	for {
		data, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		protoBuf, err := proto.Marshal(data)
		if err != nil {
			return err
		}
		err = proto.Unmarshal(protoBuf, cluster)
		if err != nil {
			return err
		}
	}
}

func (i *InfrastructureCluster) MigrateToBostionHost(ctx context.Context, cluster *biz.Cluster) error {
	grpconn, err := coonGrpc(ctx, i.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	stream, err := clusterApi.NewClusterInterfaceClient(grpconn.Conn).MigrateToBostionHost(ctx, cluster)
	if err != nil {
		return err
	}
	for {
		data, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		protoBuf, err := proto.Marshal(data)
		if err != nil {
			return err
		}
		err = proto.Unmarshal(protoBuf, cluster)
		if err != nil {
			return err
		}
	}
}

func (i *InfrastructureCluster) GetNodesSystemInfo(ctx context.Context, cluster *biz.Cluster) error {
	grpconn, err := coonGrpc(ctx, i.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	stream, err := clusterApi.NewClusterInterfaceClient(grpconn.Conn).GetNodesSystemInfo(ctx, cluster)
	if err != nil {
		return err
	}
	for {
		data, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		protoBuf, err := proto.Marshal(data)
		if err != nil {
			return err
		}
		err = proto.Unmarshal(protoBuf, cluster)
		if err != nil {
			return err
		}
	}
}

func (i *InfrastructureCluster) Install(ctx context.Context, cluster *biz.Cluster) error {
	grpconn, err := coonGrpc(ctx, i.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	stream, err := clusterApi.NewClusterInterfaceClient(grpconn.Conn).Install(ctx, cluster)
	if err != nil {
		return err
	}
	for {
		data, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		protoBuf, err := proto.Marshal(data)
		if err != nil {
			return err
		}
		err = proto.Unmarshal(protoBuf, cluster)
		if err != nil {
			return err
		}
	}
}

func (i *InfrastructureCluster) UnInstall(ctx context.Context, cluster *biz.Cluster) error {
	grpconn, err := coonGrpc(ctx, i.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	stream, err := clusterApi.NewClusterInterfaceClient(grpconn.Conn).UnInstall(ctx, cluster)
	if err != nil {
		return err
	}
	for {
		data, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		protoBuf, err := proto.Marshal(data)
		if err != nil {
			return err
		}
		err = proto.Unmarshal(protoBuf, cluster)
		if err != nil {
			return err
		}
	}
}

func (i *InfrastructureCluster) HandlerNodes(ctx context.Context, cluster *biz.Cluster) error {
	grpconn, err := coonGrpc(ctx, i.conf)
	if err != nil {
		return err
	}
	defer grpconn.Close()
	stream, err := clusterApi.NewClusterInterfaceClient(grpconn.Conn).HandlerNodes(ctx, cluster)
	if err != nil {
		return err
	}
	for {
		data, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		protoBuf, err := proto.Marshal(data)
		if err != nil {
			return err
		}
		err = proto.Unmarshal(protoBuf, cluster)
		if err != nil {
			return err
		}
	}
}

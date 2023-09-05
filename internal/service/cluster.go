package service

import (
	"context"
	"errors"
	v1 "ocean/api/cluster/v1"
	"ocean/internal/biz"

	"github.com/golang/protobuf/ptypes/empty"
)

type ClusterService struct {
	v1.UnimplementedClusterServer
	uc *biz.ClusterUsecase
}

func NewClusterService(uc *biz.ClusterUsecase) *ClusterService {
	return &ClusterService{uc: uc}
}

func (c *ClusterService) GetClusterConfig(ctx context.Context, param *v1.GetClusterConfigRequest) (*v1.GetClusterConfigResponse, error) {
	if param.Module == "" {
		return nil, errors.New("ErrorReason_MODULE_IS_EMPTY")
	}

	config, err := c.uc.GetClusterConfig(ctx, param.Module)
	if err != nil {
		return nil, err
	}
	return &v1.GetClusterConfigResponse{
		Data: string(config),
	}, nil
}

func (c *ClusterService) SaveClusterConfig(ctx context.Context, param *v1.SaveClusterConfigRequest) (*v1.Msg, error) {
	if param.Module == "" {
		return nil, errors.New("ErrorReason_MODULE_IS_EMPTY")
	}

	err := c.uc.SaveClusterConfig(ctx, param.Module, []byte(param.Data))
	if err != nil {
		return &v1.Msg{Reason: v1.ErrorReason_FAILED}, err
	}
	return &v1.Msg{Reason: v1.ErrorReason_SUCCEED}, nil
}

func (c *ClusterService) DeployCluster(ctx context.Context, _ *empty.Empty) (*v1.Msg, error) {
	err := c.uc.DeployCluster(ctx)
	if err != nil {
		return &v1.Msg{Reason: v1.ErrorReason_FAILED}, err
	}
	return &v1.Msg{Reason: v1.ErrorReason_SUCCEED}, nil
}

func (c *ClusterService) DestroyCluster(ctx context.Context, _ *empty.Empty) (*v1.Msg, error) {
	err := c.uc.DestroyCluster(ctx)
	if err != nil {
		return &v1.Msg{Reason: v1.ErrorReason_FAILED}, err
	}
	return &v1.Msg{Reason: v1.ErrorReason_SUCCEED}, nil
}

func (c *ClusterService) SyncConfigCluster(ctx context.Context, _ *empty.Empty) (*v1.Msg, error) {
	err := c.uc.SyncConfigCluster(ctx)
	if err != nil {
		return &v1.Msg{Reason: v1.ErrorReason_FAILED}, err
	}
	return &v1.Msg{Reason: v1.ErrorReason_SUCCEED}, nil
}

func (c *ClusterService) GetCluster(ctx context.Context, _ *empty.Empty) (*v1.Servers, error) {
	servers, err := c.uc.GetCluster(ctx)
	if err != nil {
		return nil, err
	}
	data := &v1.Servers{
		ClusterName: servers.ClusterName,
		Nodes:       make([]*v1.Node, 0),
	}
	for _, node := range servers.Nodes {
		data.Nodes = append(data.Nodes, &v1.Node{
			Host:         node.Host,
			Name:         node.Name,
			User:         node.User,
			Password:     node.Password,
			SudoPassword: node.SudoPassword,
			Role:         node.Role,
		})
	}

	return data, nil
}

func (c *ClusterService) SaveCluster(ctx context.Context, cluster *v1.Servers) (*v1.Msg, error) {
	if cluster.ClusterName == "" {
		return nil, errors.New("ErrorReason_CLUSTER_NAME_IS_EMPTY")
	}
	if len(cluster.Nodes) == 0 {
		return nil, errors.New("ErrorReason_NODES_IS_EMPTY")
	}
	servers := &biz.Cluster{
		ClusterName: cluster.ClusterName,
		Nodes:       make([]biz.Node, 0),
	}
	for _, node := range cluster.Nodes {
		servers.Nodes = append(servers.Nodes, biz.Node{
			Host:         node.Host,
			Name:         node.Name,
			User:         node.User,
			Password:     node.Password,
			SudoPassword: node.SudoPassword,
			Role:         node.Role,
		})
	}
	err := c.uc.SaveCluster(ctx, servers)
	if err != nil {
		return &v1.Msg{Reason: v1.ErrorReason_FAILED}, err
	}
	return &v1.Msg{Reason: v1.ErrorReason_SUCCEED}, nil
}

func (c *ClusterService) SetClusterAuth(ctx context.Context, _ *empty.Empty) (*v1.Msg, error) {
	err := c.uc.SetClusterAuth(ctx)
	if err != nil {
		return &v1.Msg{Reason: v1.ErrorReason_FAILED}, err
	}
	return &v1.Msg{Reason: v1.ErrorReason_SUCCEED}, nil
}

func (c *ClusterService) SetUpClusterTool(ctx context.Context, _ *empty.Empty) (*v1.Msg, error) {
	err := c.uc.SetUpClusterTool(ctx)
	if err != nil {
		return &v1.Msg{Reason: v1.ErrorReason_FAILED}, err
	}
	return &v1.Msg{Reason: v1.ErrorReason_SUCCEED}, nil
}

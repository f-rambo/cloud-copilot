package interfaces

import (
	"context"
	"errors"

	v1alpha1 "github.com/f-rambo/ocean/api/cluster/v1alpha1"
	"github.com/f-rambo/ocean/internal/biz"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ClusterInterface struct {
	v1alpha1.UnimplementedClusterInterfaceServer
	uc *biz.ClusterUsecase
}

func NewClusterInterface(uc *biz.ClusterUsecase) *ClusterInterface {
	return &ClusterInterface{uc: uc}
}

func (c *ClusterInterface) Ping(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.Msg, error) {
	return &v1alpha1.Msg{Message: "pong"}, nil
}

func (c *ClusterInterface) Get(ctx context.Context, clusterID *v1alpha1.ClusterID) (*v1alpha1.Cluster, error) {
	if clusterID.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	cluster, err := c.uc.Get(ctx, clusterID.Id)
	if err != nil {
		return nil, err
	}
	data := &v1alpha1.Cluster{}
	if cluster == nil {
		return data, nil
	}
	data = c.bizCLusterToCluster(cluster)
	return data, nil
}

func (c *ClusterInterface) Save(ctx context.Context, cluster *v1alpha1.Cluster) (*v1alpha1.Cluster, error) {
	if cluster.Name == "" {
		return nil, errors.New("cluster name is required")
	}
	if cluster.ApiServerAddress == "" {
		return nil, errors.New("api server address is required")
	}
	bizCluster := c.clusterToBizCluster(cluster)
	err := c.uc.Save(ctx, bizCluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (c *ClusterInterface) List(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.ClusterList, error) {
	clusters, err := c.uc.List(ctx)
	if err != nil {
		return nil, err
	}
	data := &v1alpha1.ClusterList{}
	for _, v := range clusters {
		data.Clusters = append(data.Clusters, c.bizCLusterToCluster(v))
	}
	return data, nil
}

func (c *ClusterInterface) Delete(ctx context.Context, clusterID *v1alpha1.ClusterID) (*v1alpha1.Msg, error) {
	if clusterID.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	err := c.uc.Delete(ctx, clusterID.Id)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.Msg{}, nil
}

func (c *ClusterInterface) bizCLusterToCluster(bizCluster *biz.Cluster) *v1alpha1.Cluster {
	cluster := &v1alpha1.Cluster{
		Id:               bizCluster.ID,
		Name:             bizCluster.Name,
		ServerVersion:    bizCluster.ServerVersion,
		ApiServerAddress: bizCluster.ApiServerAddress,
		Config:           bizCluster.Config,
		Addons:           bizCluster.Addons,
	}
	for _, node := range bizCluster.Nodes {
		cluster.Nodes = append(cluster.Nodes, c.bizNodeToNode(node))
	}
	return cluster
}

func (c *ClusterInterface) bizNodeToNode(bizNode *biz.Node) *v1alpha1.Node {
	node := &v1alpha1.Node{
		Id:           bizNode.ID,
		Name:         bizNode.Name,
		Labels:       bizNode.Labels,
		Annotations:  bizNode.Annotations,
		OsImage:      bizNode.OSImage,
		Kernel:       bizNode.Kernel,
		Container:    bizNode.Container,
		Kubelet:      bizNode.Kubelet,
		KubeProxy:    bizNode.KubeProxy,
		InternalIp:   bizNode.InternalIP,
		ExternalIp:   bizNode.ExternalIP,
		User:         bizNode.User,
		Password:     bizNode.Password,
		SudoPassword: bizNode.SudoPassword,
		Role:         bizNode.Role,
		ClusterId:    bizNode.ClusterID,
	}
	return node
}

func (c *ClusterInterface) clusterToBizCluster(cluster *v1alpha1.Cluster) *biz.Cluster {
	bizCluster := &biz.Cluster{
		ID:               cluster.Id,
		Name:             cluster.Name,
		ServerVersion:    cluster.ServerVersion,
		ApiServerAddress: cluster.ApiServerAddress,
		Config:           cluster.Config,
		Addons:           cluster.Addons,
	}
	for _, node := range cluster.Nodes {
		bizCluster.Nodes = append(bizCluster.Nodes, c.nodeToBizNode(node))
	}
	return bizCluster
}

func (c *ClusterInterface) nodeToBizNode(node *v1alpha1.Node) *biz.Node {
	bizNode := &biz.Node{
		ID:           node.Id,
		Name:         node.Name,
		Labels:       node.Labels,
		Annotations:  node.Annotations,
		OSImage:      node.OsImage,
		Kernel:       node.Kernel,
		Container:    node.Container,
		Kubelet:      node.Kubelet,
		KubeProxy:    node.KubeProxy,
		InternalIP:   node.InternalIp,
		ExternalIP:   node.ExternalIp,
		User:         node.User,
		Password:     node.Password,
		SudoPassword: node.SudoPassword,
		Role:         node.Role,
		ClusterID:    node.ClusterId,
	}
	return bizNode
}

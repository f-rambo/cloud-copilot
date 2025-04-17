package interfaces

import (
	"context"

	"github.com/f-rambo/cloud-copilot/api/cluster/v1alpha1"
	"github.com/f-rambo/cloud-copilot/api/common"
	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/emptypb"
)

type ClusterInterface struct {
	v1alpha1.UnimplementedClusterInterfaceServer
	clusterUc *biz.ClusterUsecase
	c         *conf.Bootstrap
	log       *log.Helper
}

func NewClusterInterface(clusterUc *biz.ClusterUsecase, c *conf.Bootstrap, logger log.Logger) *ClusterInterface {
	return &ClusterInterface{
		clusterUc: clusterUc,
		c:         c,
		log:       log.NewHelper(logger),
	}
}

func (c *ClusterInterface) Ping(ctx context.Context, _ *emptypb.Empty) (*common.Msg, error) {
	return common.Response(), nil
}

func (c *ClusterInterface) GetCluster(ctx context.Context, clusterId int64) (*biz.Cluster, error) {
	return c.clusterUc.Get(ctx, clusterId)
}

func (c *ClusterInterface) GetClusterProviders(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.ClusterProviders, error) {
	clusterTypes := c.clusterUc.GetClusterProviders()
	clusterTypeResponse := &v1alpha1.ClusterProviders{ClusterProviders: make([]*v1alpha1.ClusterProvider, 0)}
	for _, clusterType := range clusterTypes {
		clusterTypeResponse.ClusterProviders = append(clusterTypeResponse.ClusterProviders, &v1alpha1.ClusterProvider{
			Id:      int32(clusterType),
			Name:    clusterType.String(),
			IsCloud: clusterType.IsCloud(),
		})
	}
	return clusterTypeResponse, nil
}

func (c *ClusterInterface) GetClusterStatuses(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.ClusterStatuses, error) {
	clusterStatuses := c.clusterUc.GetClusterStatus()
	clusterStatusResponse := &v1alpha1.ClusterStatuses{ClusterStatuses: make([]*v1alpha1.ClusterStatus, 0)}
	for _, clusterStatus := range clusterStatuses {
		clusterStatusResponse.ClusterStatuses = append(clusterStatusResponse.ClusterStatuses, &v1alpha1.ClusterStatus{
			Id:   int32(clusterStatus),
			Name: clusterStatus.String(),
		})
	}
	return clusterStatusResponse, nil
}

func (c *ClusterInterface) GetClusterLevels(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.ClusterLevels, error) {
	clusterLevels := c.clusterUc.GetClusterLevels()
	clusterLevelResponse := &v1alpha1.ClusterLevels{ClusterLevels: make([]*v1alpha1.ClusterLevel, 0)}
	for _, clusterLevel := range clusterLevels {
		clusterLevelResponse.ClusterLevels = append(clusterLevelResponse.ClusterLevels, &v1alpha1.ClusterLevel{
			Id:   int32(clusterLevel),
			Name: clusterLevel.String(),
		})
	}
	return clusterLevelResponse, nil
}

func (c *ClusterInterface) GetNodeRoles(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.NodeRoles, error) {
	nodeRoles := c.clusterUc.GetNodeRoles()
	nodeRoleResponse := &v1alpha1.NodeRoles{NodeRoles: make([]*v1alpha1.NodeRole, 0)}
	for _, nodeRole := range nodeRoles {
		nodeRoleResponse.NodeRoles = append(nodeRoleResponse.NodeRoles, &v1alpha1.NodeRole{
			Id:   int32(nodeRole),
			Name: nodeRole.String(),
		})
	}
	return nodeRoleResponse, nil
}

func (c *ClusterInterface) GetNodeStatuses(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.NodeStatuses, error) {
	nodeStatuses := c.clusterUc.GetNodeStatuses()
	nodeStatusResponse := &v1alpha1.NodeStatuses{NodeStatuses: make([]*v1alpha1.NodeStatus, 0)}
	for _, nodeStatus := range nodeStatuses {
		nodeStatusResponse.NodeStatuses = append(nodeStatusResponse.NodeStatuses, &v1alpha1.NodeStatus{
			Id:   int32(nodeStatus),
			Name: nodeStatus.String(),
		})
	}
	return nodeStatusResponse, nil
}

func (c *ClusterInterface) GetNodeGroupTypes(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.NodeGroupTypes, error) {
	nodeGroupTypes := c.clusterUc.GetNodeGroupTypes()
	nodeGroupTypeResponse := &v1alpha1.NodeGroupTypes{NodeGroupTypes: make([]*v1alpha1.NodeGroupType, 0)}
	for _, nodeGroupType := range nodeGroupTypes {
		nodeGroupTypeResponse.NodeGroupTypes = append(nodeGroupTypeResponse.NodeGroupTypes, &v1alpha1.NodeGroupType{
			Id:   int32(nodeGroupType),
			Name: nodeGroupType.String(),
		})
	}
	return nodeGroupTypeResponse, nil
}

func (c *ClusterInterface) GetResourceTypes(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.ResourceTypes, error) {
	resourceTypes := c.clusterUc.GetResourceTypes()
	resourceTypeResponse := &v1alpha1.ResourceTypes{ResourceTypes: make([]*v1alpha1.ResourceType, 0)}
	for _, resourceType := range resourceTypes {
		resourceTypeResponse.ResourceTypes = append(resourceTypeResponse.ResourceTypes, &v1alpha1.ResourceType{
			Id:   int32(resourceType),
			Name: resourceType.String(),
		})
	}
	return resourceTypeResponse, nil
}

func (c *ClusterInterface) Get(ctx context.Context, clusterArgs *v1alpha1.ClusterArgs) (*v1alpha1.Cluster, error) {
	if clusterArgs.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	cluster, err := c.clusterUc.Get(ctx, clusterArgs.Id)
	if err != nil {
		return nil, err
	}
	return c.bizCLusterToCluster(cluster), nil
}
func (c *ClusterInterface) Save(ctx context.Context, clusterArgs *v1alpha1.ClusterSaveArgs) (*v1alpha1.Cluster, error) {
	if clusterArgs.Name == "" || clusterArgs.PrivateKey == "" || clusterArgs.Provider == "" || clusterArgs.PublicKey == "" {
		return nil, errors.New("cluster name, private key, type and public key are required")
	}
	if biz.ClusterProviderFromString(clusterArgs.Provider) == 0 {
		return nil, errors.New("cluster type is invalid")
	}
	if biz.ClusterProviderFromString(clusterArgs.Provider).IsCloud() && (clusterArgs.AccessId == "" || clusterArgs.AccessKey == "" || clusterArgs.Region == "") {
		return nil, errors.New("access key id and secret access key, region are required")
	}
	if !biz.ClusterProviderFromString(clusterArgs.Provider).IsCloud() && (clusterArgs.NodeUsername == "" || clusterArgs.NodeStartIp == "" || clusterArgs.NodeEndIp == "") {
		return nil, errors.New("node username, start ip and end ip are required")
	}
	if clusterArgs.Id != 0 {
		clusterRes, err := c.clusterUc.Get(ctx, clusterArgs.Id)
		if err != nil {
			return nil, err
		}
		if clusterRes == nil || clusterRes.Id == 0 {
			return nil, errors.New("cluster not found")
		}
	} else {
		clusterRes, err := c.clusterUc.GetByName(ctx, clusterArgs.Name)
		if err != nil {
			return nil, err
		}
		if !clusterRes.IsEmpty() {
			return nil, errors.New("cluster already exists")
		}
	}
	cluster := &biz.Cluster{
		Id:              clusterArgs.Id,
		Name:            clusterArgs.Name,
		Provider:        biz.ClusterProviderFromString(clusterArgs.Provider),
		PublicKey:       clusterArgs.PublicKey,
		PrivateKey:      clusterArgs.PrivateKey,
		AccessId:        clusterArgs.AccessId,
		AccessKey:       clusterArgs.AccessKey,
		Region:          clusterArgs.Region,
		DefaultUsername: clusterArgs.NodeUsername,
		NodeStartIp:     clusterArgs.NodeStartIp,
		NodeEndIp:       clusterArgs.NodeEndIp,
		UserId:          biz.GetUserInfo(ctx).Id,
	}
	err := c.clusterUc.Save(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return c.bizCLusterToCluster(cluster), nil
}

func (c *ClusterInterface) Start(ctx context.Context, clusterArgs *v1alpha1.ClusterArgs) (*common.Msg, error) {
	if clusterArgs.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	err := c.clusterUc.StartCluster(ctx, clusterArgs.Id)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (c *ClusterInterface) Stop(ctx context.Context, clusterArgs *v1alpha1.ClusterArgs) (*common.Msg, error) {
	if clusterArgs.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	err := c.clusterUc.StopCluster(ctx, clusterArgs.Id)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (c *ClusterInterface) List(ctx context.Context, clusterArgs *v1alpha1.ClusterArgs) (*v1alpha1.ClusterList, error) {
	data := &v1alpha1.ClusterList{}
	clusters, total, err := c.clusterUc.List(ctx, clusterArgs.Name, clusterArgs.Page, clusterArgs.PageSize)
	if err != nil {
		return nil, err
	}
	for _, v := range clusters {
		data.Clusters = append(data.Clusters, c.bizCLusterToCluster(v))
	}
	data.Total = total
	return data, nil
}

func (c *ClusterInterface) Delete(ctx context.Context, clusterArgs *v1alpha1.ClusterArgs) (*common.Msg, error) {
	if clusterArgs.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	err := c.clusterUc.Delete(ctx, clusterArgs.Id)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (c *ClusterInterface) GetRegions(ctx context.Context, clusterArgs *v1alpha1.ClusterRegionArgs) (*v1alpha1.Regions, error) {
	if clusterArgs.Provider == "" || clusterArgs.AccessId == "" || clusterArgs.AccessKey == "" {
		return nil, errors.New("Provider, access id and access key are required")
	}
	regions, err := c.clusterUc.GetRegions(ctx, biz.ClusterProviderFromString(clusterArgs.Provider), clusterArgs.AccessId, clusterArgs.AccessKey)
	if err != nil {
		return nil, err
	}
	data := make([]*v1alpha1.Region, 0)
	for _, v := range regions {
		data = append(data, &v1alpha1.Region{
			Id:   v.RefId,
			Name: v.Name,
		})
	}
	return &v1alpha1.Regions{Regions: data}, nil
}

func (c *ClusterInterface) bizCLusterToCluster(bizCluster *biz.Cluster) *v1alpha1.Cluster {
	nodes := make([]*v1alpha1.Node, 0)
	for _, v := range bizCluster.Nodes {
		if v == nil {
			continue
		}
		nodes = append(nodes, c.bizNodeToNode(v))
	}
	nodeGroups := make([]*v1alpha1.NodeGroup, 0)
	for _, v := range bizCluster.NodeGroups {
		if v == nil {
			continue
		}
		nodeGroups = append(nodeGroups, c.bizNodeGroupToNodeGroup(v))
	}
	return &v1alpha1.Cluster{
		Id:               bizCluster.Id,
		Name:             bizCluster.Name,
		ApiServerAddress: bizCluster.ApiServerAddress,
		Status:           bizCluster.Status.String(),
		Domain:           bizCluster.Domain,
		NodeNumber:       int32(len(bizCluster.Nodes)),
		Provider:         bizCluster.Provider.String(),
		PublicKey:        bizCluster.PublicKey,
		PrivateKey:       bizCluster.PrivateKey,
		Region:           bizCluster.Region,
		AccessId:         bizCluster.AccessId,
		AccessKey:        bizCluster.AccessKey,
		NodeUsername:     bizCluster.DefaultUsername,
		NodeStartIp:      bizCluster.NodeStartIp,
		NodeEndIp:        bizCluster.NodeEndIp,
		Nodes:            nodes,
		NodeGroups:       nodeGroups,
		ClusterResource: &v1alpha1.ClusterResource{
			Cpu:    bizCluster.GetCpuCount(),
			Gpu:    bizCluster.GetGpuCount(),
			Memory: bizCluster.GetMemoryCount(),
			Disk:   bizCluster.GetDiskSizeCount(),
		},
	}
}

func (c *ClusterInterface) bizNodeToNode(node *biz.Node) *v1alpha1.Node {
	return &v1alpha1.Node{
		Id:         node.Id,
		Ip:         node.Ip,
		Name:       node.Name,
		Role:       node.Role.String(),
		User:       node.User,
		Status:     node.Status.String(),
		InstanceId: node.InstanceId,
	}
}

func (c *ClusterInterface) bizNodeGroupToNodeGroup(nodeGroup *biz.NodeGroup) *v1alpha1.NodeGroup {
	return &v1alpha1.NodeGroup{
		Id:         nodeGroup.Id,
		Name:       nodeGroup.Name,
		Type:       nodeGroup.Type.String(),
		Os:         nodeGroup.Os,
		Arch:       nodeGroup.Arch.String(),
		Cpu:        nodeGroup.Cpu,
		Memory:     nodeGroup.Memory,
		Gpu:        nodeGroup.Gpu,
		GpuSpec:    nodeGroup.GpuSpec.String(),
		MinSize:    nodeGroup.MinSize,
		MaxSize:    nodeGroup.MaxSize,
		TargetSize: nodeGroup.TargetSize,
	}
}

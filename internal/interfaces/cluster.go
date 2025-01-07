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
	userUc    *biz.UserUseCase
	c         *conf.Bootstrap
	log       *log.Helper
}

func NewClusterInterface(clusterUc *biz.ClusterUsecase, userUc *biz.UserUseCase, c *conf.Bootstrap, logger log.Logger) *ClusterInterface {
	return &ClusterInterface{
		clusterUc: clusterUc,
		userUc:    userUc,
		c:         c,
		log:       log.NewHelper(logger),
	}
}

func (c *ClusterInterface) Ping(ctx context.Context, _ *emptypb.Empty) (*common.Msg, error) {
	return common.Response(), nil
}

func (c *ClusterInterface) GetClusterTypes(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.ClusterTypes, error) {
	clusterTypes := c.clusterUc.GetClusterTypes()
	clusterTypeResponse := &v1alpha1.ClusterTypes{ClusterTypes: make([]*v1alpha1.ClusterType, 0)}
	for _, clusterType := range clusterTypes {
		clusterTypeResponse.ClusterTypes = append(clusterTypeResponse.ClusterTypes, &v1alpha1.ClusterType{
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

func (c *ClusterInterface) Get(ctx context.Context, clusterIdArgs *v1alpha1.ClusterIdMessge) (*v1alpha1.Cluster, error) {
	if clusterIdArgs.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	cluster, err := c.clusterUc.Get(ctx, clusterIdArgs.Id)
	if err != nil {
		return nil, err
	}
	data := &v1alpha1.Cluster{}
	if cluster == nil {
		return data, nil
	}
	return c.bizCLusterToCluster(cluster), nil
}
func (c *ClusterInterface) Save(ctx context.Context, clusterArgs *v1alpha1.ClusterSaveArgs) (msg *v1alpha1.ClusterIdMessge, err error) {
	if clusterArgs.Name == "" || clusterArgs.PrivateKey == "" || clusterArgs.Type == 0 || clusterArgs.PublicKey == "" {
		return nil, errors.New("cluster name, private key, type and public key are required")
	}
	if biz.ClusterType(clusterArgs.Type).IsCloud() && (clusterArgs.AccessId == "" || clusterArgs.AccessKey == "" || clusterArgs.Region == "") {
		return nil, errors.New("access key id and secret access key, region are required")
	}
	if clusterArgs.Type == int32(biz.ClusterType_LOCAL.Number()) && (clusterArgs.NodeUsername == "" || clusterArgs.NodeStartIp == "" || clusterArgs.NodeEndIp == "") {
		return nil, errors.New("node username, start ip and end ip are required")
	}
	cluster := &biz.Cluster{}
	if clusterArgs.Id != 0 {
		cluster, err = c.clusterUc.Get(ctx, clusterArgs.Id)
		if err != nil {
			return nil, err
		}
		if cluster == nil || cluster.Id == 0 {
			return nil, errors.New("cluster not found")
		}
	}
	if cluster.Id == 0 {
		cluster, err = c.clusterUc.GetByName(ctx, clusterArgs.Name)
		if err != nil {
			return nil, err
		}
		if cluster.Id != 0 {
			return nil, errors.New("cluster already exists")
		}
	}
	cluster.Name = clusterArgs.Name
	cluster.Type = biz.ClusterType(clusterArgs.Type)
	cluster.PublicKey = clusterArgs.PublicKey
	cluster.PrivateKey = clusterArgs.PrivateKey
	cluster.AccessId = clusterArgs.AccessId
	cluster.AccessKey = clusterArgs.AccessKey
	cluster.Region = clusterArgs.Region
	cluster.NodeUser = clusterArgs.NodeUsername
	cluster.NodeStartIp = clusterArgs.NodeStartIp
	cluster.NodeEndIp = clusterArgs.NodeEndIp
	user, err := c.userUc.GetUserInfo(ctx)
	if err != nil {
		return nil, err
	}
	cluster.UserId = user.Id
	err = c.clusterUc.Save(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.ClusterIdMessge{Id: cluster.Id}, nil
}

func (c *ClusterInterface) Start(ctx context.Context, clusterArgs *v1alpha1.ClusterIdMessge) (*common.Msg, error) {
	if clusterArgs.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	err := c.clusterUc.StartCluster(ctx, clusterArgs.Id)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (c *ClusterInterface) Stop(ctx context.Context, clusterArgs *v1alpha1.ClusterIdMessge) (*common.Msg, error) {
	if clusterArgs.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	err := c.clusterUc.StopCluster(ctx, clusterArgs.Id)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

func (c *ClusterInterface) List(ctx context.Context, _ *emptypb.Empty) (*v1alpha1.ClusterList, error) {
	data := &v1alpha1.ClusterList{}
	clusters, err := c.clusterUc.List(ctx)
	if err != nil {
		return nil, err
	}
	if len(clusters) == 0 {
		return data, nil
	}
	for _, v := range clusters {
		data.Clusters = append(data.Clusters, c.bizCLusterToCluster(v))
	}
	return data, nil
}

func (c *ClusterInterface) Delete(ctx context.Context, clusterID *v1alpha1.ClusterIdMessge) (*common.Msg, error) {
	if clusterID.Id == 0 {
		return nil, errors.New("cluster id is required")
	}
	cluster, err := c.clusterUc.Get(ctx, clusterID.Id)
	if err != nil {
		return nil, err
	}
	if cluster == nil || cluster.Id == 0 {
		return nil, errors.New("cluster not found")
	}
	err = c.clusterUc.Delete(ctx, clusterID.Id)
	if err != nil {
		return nil, err
	}
	return common.Response(), nil
}

// get regions
func (c *ClusterInterface) GetRegions(ctx context.Context, clusterArgs *v1alpha1.ClusterRegionArgs) (*v1alpha1.Regions, error) {
	if clusterArgs == nil || clusterArgs.Type == 0 || clusterArgs.AccessId == "" || clusterArgs.AccessKey == "" {
		return nil, errors.New("type, access id and access key are required")
	}
	cluster := &biz.Cluster{Type: biz.ClusterType(clusterArgs.Type), AccessId: clusterArgs.AccessId, AccessKey: clusterArgs.AccessKey}
	err := c.clusterUc.GetRegions(ctx, cluster)
	if err != nil {
		return nil, err
	}
	regions := make([]*v1alpha1.Region, 0)
	for _, v := range cluster.GetCloudResource(biz.ResourceType_REGION) {
		regions = append(regions, &v1alpha1.Region{
			Id:   v.RefId,
			Name: v.Name,
		})
	}
	return &v1alpha1.Regions{Regions: regions}, nil
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
	regionName := ""
	resources := bizCluster.GetCloudResource(biz.ResourceType_REGION)
	for _, v := range resources {
		if v.RefId == bizCluster.Region {
			regionName = v.Name
		}
	}
	if regionName == "" {
		regionName = bizCluster.Region
	}
	return &v1alpha1.Cluster{
		Id:               bizCluster.Id,
		Name:             bizCluster.Name,
		Version:          bizCluster.Version,
		ApiServerAddress: bizCluster.ApiServerAddress,
		Status:           int32(bizCluster.Status),
		Type:             int32(bizCluster.Type),
		PublicKey:        bizCluster.PublicKey,
		PrivateKey:       bizCluster.PrivateKey,
		Region:           bizCluster.Region,
		RegionName:       regionName,
		AccessId:         bizCluster.AccessId,
		AccessKey:        bizCluster.AccessKey,
		CreateAt:         bizCluster.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdateAt:         bizCluster.UpdatedAt.Format("2006-01-02 15:04:05"),
		Nodes:            nodes,
		NodeGroups:       nodeGroups,
		NodeUsername:     bizCluster.NodeUser,
		NodeStartIp:      bizCluster.NodeStartIp,
		NodeEndIp:        bizCluster.NodeEndIp,
	}
}

func (c *ClusterInterface) bizNodeToNode(node *biz.Node) *v1alpha1.Node {
	return &v1alpha1.Node{
		Id:         node.Id,
		Ip:         node.Ip,
		Name:       node.Name,
		Role:       int32(node.Role),
		User:       node.User,
		Status:     int32(node.Status),
		InstanceId: node.InstanceId,
		UpdateAt:   node.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func (c *ClusterInterface) bizNodeGroupToNodeGroup(nodeGroup *biz.NodeGroup) *v1alpha1.NodeGroup {
	return &v1alpha1.NodeGroup{
		Id:         nodeGroup.Id,
		Name:       nodeGroup.Name,
		Type:       int32(nodeGroup.Type),
		Os:         nodeGroup.Os,
		Arch:       nodeGroup.Arch.String(),
		Cpu:        nodeGroup.Cpu,
		Memory:     nodeGroup.Memory,
		Gpu:        nodeGroup.Gpu,
		GpuSpec:    nodeGroup.GpuSpec.String(),
		MinSize:    nodeGroup.MinSize,
		MaxSize:    nodeGroup.MaxSize,
		TargetSize: nodeGroup.TargetSize,
		UpdateAt:   nodeGroup.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

package biz

import (
	"context"
	"fmt"
	"math"
	"sync"

	confPkg "github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type ClusterData interface {
	Save(context.Context, *Cluster) error
	Get(context.Context, int64) (*Cluster, error)
	GetByName(context.Context, string) (*Cluster, error)
	List(context.Context, *Cluster) ([]*Cluster, error)
	Delete(context.Context, int64) error
}

type ClusterInfrastructure interface {
	GetRegions(context.Context, *Cluster) error
	GetZones(context.Context, *Cluster) error
	CreateCloudBasicResource(context.Context, *Cluster) error
	DeleteCloudBasicResource(context.Context, *Cluster) error
	ManageNodeResource(context.Context, *Cluster) error
	GetNodesSystemInfo(context.Context, *Cluster) error
	Install(context.Context, *Cluster) error
	UnInstall(context.Context, *Cluster) error
	HandlerNodes(context.Context, *Cluster) error
}

type ClusterRuntime interface {
	CurrentCluster(context.Context, *Cluster) error
	HandlerNodes(context.Context, *Cluster) error
	MigrateToCluster(context.Context, *Cluster) error
}

type ClusterAgent interface {
}

type ClusterUsecase struct {
	clusterData           ClusterData
	clusterInfrastructure ClusterInfrastructure
	clusterRuntime        ClusterRuntime
	locks                 map[int64]*sync.Mutex
	locksMux              sync.Mutex
	eventChan             chan *Cluster
	conf                  *confPkg.Bootstrap
	log                   *log.Helper
}

func NewClusterUseCase(conf *confPkg.Bootstrap, clusterData ClusterData, clusterInfrastructure ClusterInfrastructure, clusterRuntime ClusterRuntime, logger log.Logger) *ClusterUsecase {
	return &ClusterUsecase{
		clusterData:           clusterData,
		clusterInfrastructure: clusterInfrastructure,
		clusterRuntime:        clusterRuntime,
		conf:                  conf,
		log:                   log.NewHelper(logger),
		locks:                 make(map[int64]*sync.Mutex),
		eventChan:             make(chan *Cluster, ClusterPoolNumber),
	}
}

func (c *Cluster) GetCloudResource(resourceType ResourceType) []*CloudResource {
	cloudResources := make([]*CloudResource, 0)
	for _, resources := range c.CloudResources {
		if resources != nil && resources.Type == resourceType {
			cloudResources = append(cloudResources, resources)
		}
	}
	return cloudResources
}

func (c *Cluster) AddCloudResource(resource *CloudResource) {
	if resource == nil {
		return
	}
	if c.CloudResources == nil {
		c.CloudResources = make([]*CloudResource, 0)
	}
	if resource.Id == "" {
		resource.Id = uuid.New().String()
	}
	c.CloudResources = append(c.CloudResources, resource)
}

func (c *Cluster) DeleteCloudResource(resourceType ResourceType) {
	cloudResources := make([]*CloudResource, 0)
	for _, resources := range c.CloudResources {
		if resources.Type != resourceType {
			cloudResources = append(cloudResources, resources)
		}
	}
	c.CloudResources = cloudResources
}

func (c *Cluster) SettingClusterAvailabilityZone(clusterLevel *confPkg.Level) {
	var maxNodeNumber int32 = 0
	for _, nodeGroup := range c.NodeGroups {
		maxNodeNumber += nodeGroup.TargetSize
	}
	if maxNodeNumber < clusterLevel.Basic {
		c.Level = ClusterLevel_BASIC
	}
	if maxNodeNumber < clusterLevel.Advanced && maxNodeNumber >= clusterLevel.Basic {
		c.Level = ClusterLevel_ADVANCED
	}
	if maxNodeNumber >= clusterLevel.Advanced {
		c.Level = ClusterLevel_STANDARD
	}
	zones := c.GetCloudResource(ResourceType_AVAILABILITY_ZONES)
	if len(zones) == 0 {
		return
	}
	c.DeleteCloudResource(ResourceType_AVAILABILITY_ZONES)
	zoneNumber := len(zones)
	if c.Level == ClusterLevel_BASIC {
		zoneNumber = 1
	}
	if c.Level == ClusterLevel_STANDARD {
		zoneNumber = int(math.Ceil(float64(zoneNumber) / 2))
	}
	for _, zone := range zones[:zoneNumber] {
		c.AddCloudResource(zone)
	}
}

func (c *Cluster) SettingDefaultNodeGroup(nodegroupConfig *confPkg.NodeGroupConfig) {
	if c.NodeGroups == nil {
		c.NodeGroups = make([]*NodeGroup, 0)
	}
	if len(c.NodeGroups) == 0 {
		c.NodeGroups = append(c.NodeGroups, &NodeGroup{
			Id:         uuid.NewString(),
			Name:       "default",
			ClusterId:  c.Id,
			Type:       NodeGroupType_NORMAL,
			TargetSize: nodegroupConfig.TargetSize,
			MaxSize:    nodegroupConfig.MaxSize,
			MinSize:    nodegroupConfig.MinSize,
			Arch:       NodeArchType_AMD64,
			Cpu:        nodegroupConfig.Cpu,
			Memory:     nodegroupConfig.Memory,
		})
	}
	if c.Nodes == nil {
		c.Nodes = make([]*Node, 0)
	}
	if len(c.Nodes) == 0 {
		c.Nodes = append(c.Nodes, &Node{
			Name:           "default",
			Status:         NodeStatus_NODE_FINDING,
			Role:           NodeRole_MASTER,
			NodeGroupId:    c.NodeGroups[0].Id,
			SystemDiskSize: nodegroupConfig.DiskSize,
			ClusterId:      c.Id,
		})
	}
}

func (c *Cluster) settingDefatultIngressRules(rules []*confPkg.IngressRule) {
	if c.IngressControllerRules == nil {
		c.IngressControllerRules = make([]*IngressControllerRule, 0)
	}
	for _, rule := range rules {
		clusterIngressControllerRule := &IngressControllerRule{
			StartPort: rule.StartPort,
			EndPort:   rule.EndPort,
			Protocol:  rule.Protocol,
			IpCidr:    rule.IpCidr,
			ClusterId: c.Id,
			Name:      rule.Name,
		}
		if rule.Access == confPkg.Access_Private {
			clusterIngressControllerRule.Access = IngressControllerRuleAccess_PRIVATE
		}
		if rule.Access == confPkg.Access_Public {
			clusterIngressControllerRule.Access = IngressControllerRuleAccess_PUBLIC
		}
		clusterIngressControllerRule.Id = fmt.Sprintf("%s-%s-%s-%d-%d-%d-%d",
			clusterIngressControllerRule.Name,
			clusterIngressControllerRule.Protocol,
			clusterIngressControllerRule.IpCidr,
			clusterIngressControllerRule.StartPort,
			clusterIngressControllerRule.EndPort,
			clusterIngressControllerRule.Access,
			clusterIngressControllerRule.ClusterId,
		)
		clusterIngressControllerRule.Id = utils.Md5(clusterIngressControllerRule.Id)
		c.IngressControllerRules = append(c.IngressControllerRules, clusterIngressControllerRule)
	}
}

func (c ClusterType) IsCloud() bool {
	return c != ClusterType_LOCAL
}

func (ng *NodeGroup) SetTargetSize(size int32) {
	ng.TargetSize = size
}

func (c *Cluster) GetNodeGroup(nodeGroupId string) *NodeGroup {
	for _, nodeGroup := range c.NodeGroups {
		if nodeGroup.Id == nodeGroupId {
			return nodeGroup
		}
	}
	return nil
}

func (n *Node) SetNodeStatus(status NodeStatus) {
	n.Status = status
}

func (uc *ClusterUsecase) GetClusterStatus() []ClusterStatus {
	clusterStatus := make([]ClusterStatus, 0)
	for _, v := range ClusterStatus_value {
		if ClusterStatus(v) == ClusterStatus_UNSPECIFIED {
			continue
		}
		clusterStatus = append(clusterStatus, ClusterStatus(v))
	}
	return clusterStatus
}

func (uc *ClusterUsecase) GetClusterTypes() []ClusterType {
	clusterTypes := make([]ClusterType, 0)
	for _, v := range ClusterType_value {
		if ClusterType(v) == ClusterType_ClusterType_UNSPECIFIED {
			continue
		}
		clusterTypes = append(clusterTypes, ClusterType(v))
	}
	return clusterTypes
}

func (uc *ClusterUsecase) GetClusterLevels() []ClusterLevel {
	clusterLevels := make([]ClusterLevel, 0)
	for _, v := range ClusterLevel_value {
		if ClusterLevel(v) == ClusterLevel_ClusterLevel_UNSPECIFIED {
			continue
		}
		clusterLevels = append(clusterLevels, ClusterLevel(v))
	}
	return clusterLevels
}

func (uc *ClusterUsecase) GetNodeRoles() []NodeRole {
	nodeRoles := make([]NodeRole, 0)
	for _, v := range NodeRole_value {
		if NodeRole(v) == NodeRole_NodeRole_UNSPECIFIED {
			continue
		}
		nodeRoles = append(nodeRoles, NodeRole(v))
	}
	return nodeRoles
}

func (uc *ClusterUsecase) GetNodeStatuses() []NodeStatus {
	nodeStatuses := make([]NodeStatus, 0)
	for _, v := range NodeStatus_value {
		if NodeStatus(v) == NodeStatus_NodeStatus_UNSPECIFIED {
			continue
		}
		nodeStatuses = append(nodeStatuses, NodeStatus(v))
	}
	return nodeStatuses
}

func (uc *ClusterUsecase) GetNodeGroupTypes() []NodeGroupType {
	nodeGroupTypes := make([]NodeGroupType, 0)
	for _, v := range NodeGroupType_value {
		if NodeGroupType(v) == NodeGroupType_NodeGroupType_UNSPECIFIED {
			continue
		}
		nodeGroupTypes = append(nodeGroupTypes, NodeGroupType(v))
	}
	return nodeGroupTypes
}

func (uc *ClusterUsecase) GetResourceTypes() []ResourceType {
	resourceTypes := make([]ResourceType, 0)
	for _, v := range ResourceType_value {
		if ResourceType(v) == ResourceType_RESOURCE_TYPE_UNSPECIFIED {
			continue
		}
		resourceTypes = append(resourceTypes, ResourceType(v))
	}
	return resourceTypes
}

func (uc *ClusterUsecase) Get(ctx context.Context, id int64) (*Cluster, error) {
	cluster, err := uc.clusterData.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (uc *ClusterUsecase) GetByName(ctx context.Context, name string) (*Cluster, error) {
	cluster, err := uc.clusterData.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (uc *ClusterUsecase) List(ctx context.Context) ([]*Cluster, error) {
	return uc.clusterData.List(ctx, nil)
}

func (uc *ClusterUsecase) Delete(ctx context.Context, clusterID int64) error {
	return uc.clusterData.Delete(ctx, clusterID)
}

func (uc *ClusterUsecase) Save(ctx context.Context, cluster *Cluster) error {
	if cluster.Level.String() == "" {
		cluster.Level = ClusterLevel_BASIC
	}
	return uc.clusterData.Save(ctx, cluster)
}

func (uc *ClusterUsecase) GetRegions(ctx context.Context, cluster *Cluster) ([]*CloudResource, error) {
	if cluster.Type == ClusterType_LOCAL {
		return []*CloudResource{}, nil
	}
	err := uc.clusterInfrastructure.GetZones(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return cluster.GetCloudResource(ResourceType_AVAILABILITY_ZONES), nil
}

func (uc *ClusterUsecase) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case data, ok := <-uc.eventChan:
			if !ok {
				return nil
			}
			err := uc.handleEvent(ctx, data)
			if err != nil {
				uc.log.Errorf("cluster handle event error: %v", err)
			}
		}
	}
}

func (uc *ClusterUsecase) Stop(ctx context.Context) error {
	close(uc.eventChan)
	return nil
}

func (uc *ClusterUsecase) Apply(cluster *Cluster) error {
	if cluster == nil || cluster.Id == 0 {
		return errors.New("invalid cluster")
	}
	if uc.eventChan == nil {
		return errors.New("cluster event channel is nil")
	}
	select {
	case uc.eventChan <- cluster:
		return nil
	default:
		return errors.New("cluster event channel is either full or closed")
	}
}

func (uc *ClusterUsecase) getLock(clusterID int64) *sync.Mutex {
	uc.locksMux.Lock()
	defer uc.locksMux.Unlock()

	if clusterID < 0 {
		uc.log.Errorf("Invalid clusterID: %d", clusterID)
		return &sync.Mutex{}
	}

	if _, exists := uc.locks[clusterID]; !exists {
		uc.locks[clusterID] = &sync.Mutex{}
	}
	return uc.locks[clusterID]
}

func (uc *ClusterUsecase) handleEvent(ctx context.Context, cluster *Cluster) (err error) {
	lock := uc.getLock(cluster.Id)
	lock.Lock()
	defer func() {
		if err != nil {
			return
		}
		lock.Unlock()
		err = uc.clusterData.Save(ctx, cluster)
	}()
	if cluster.DeletedAt.Valid {
		for _, node := range cluster.Nodes {
			if node.Status == NodeStatus_NodeStatus_UNSPECIFIED || node.Status == NodeStatus_NODE_DELETED {
				continue
			}
			node.SetNodeStatus(NodeStatus_NODE_DELETING)
		}
		err = uc.clusterRuntime.HandlerNodes(ctx, cluster)
		if err != nil {
			return err
		}
		err = uc.clusterInfrastructure.HandlerNodes(ctx, cluster)
		if err != nil {
			return err
		}
		err = uc.clusterInfrastructure.UnInstall(ctx, cluster)
		if err != nil {
			return err
		}
		err = uc.clusterInfrastructure.ManageNodeResource(ctx, cluster)
		if err != nil {
			return err
		}
		err = uc.clusterInfrastructure.DeleteCloudBasicResource(ctx, cluster)
		if err != nil {
			return err
		}
		for _, node := range cluster.Nodes {
			node.SetNodeStatus(NodeStatus_NODE_DELETED)
		}
		return nil
	}
	err = uc.clusterRuntime.CurrentCluster(ctx, cluster)
	if errors.Is(err, ErrClusterNotFound) {
		return uc.handlerClusterNotInstalled(ctx, cluster)
	}
	if err != nil {
		return err
	}
	err = uc.clusterInfrastructure.GetNodesSystemInfo(ctx, cluster)
	if err != nil {
		return err
	}
	for _, node := range cluster.Nodes {
		if node.Status == NodeStatus_NODE_FINDING {
			node.SetNodeStatus(NodeStatus_NODE_CREATING)
		}
	}
	err = uc.clusterRuntime.HandlerNodes(ctx, cluster)
	if err != nil {
		return err
	}
	err = uc.clusterInfrastructure.ManageNodeResource(ctx, cluster)
	if err != nil {
		return err
	}
	err = uc.clusterInfrastructure.HandlerNodes(ctx, cluster)
	if err != nil {
		return err
	}
	for _, node := range cluster.Nodes {
		if node.Status == NodeStatus_NODE_CREATING {
			node.SetNodeStatus(NodeStatus_NODE_RUNNING)
		}
	}
	return
}

func (uc *ClusterUsecase) handlerClusterNotInstalled(ctx context.Context, cluster *Cluster) error {
	cluster.SettingDefaultNodeGroup(uc.conf.Cluster.NodegroupConfig)
	cluster.settingDefatultIngressRules(uc.conf.Cluster.IngressRules)
	if cluster.Type.IsCloud() {
		err := uc.clusterInfrastructure.GetRegions(ctx, cluster)
		if err != nil {
			return err
		}
		err = uc.clusterInfrastructure.GetZones(ctx, cluster)
		if err != nil {
			return err
		}
		cluster.SettingClusterAvailabilityZone(uc.conf.Cluster.Level)
		err = uc.clusterInfrastructure.CreateCloudBasicResource(ctx, cluster)
		if err != nil {
			return err
		}
	}
	err := uc.clusterInfrastructure.GetNodesSystemInfo(ctx, cluster)
	if err != nil {
		return err
	}
	for _, node := range cluster.Nodes {
		if node.Status == NodeStatus_NODE_FINDING {
			node.SetNodeStatus(NodeStatus_NODE_CREATING)
		}
	}
	err = uc.clusterInfrastructure.ManageNodeResource(ctx, cluster)
	if err != nil {
		return err
	}
	err = uc.clusterInfrastructure.Install(ctx, cluster)
	if err != nil {
		return err
	}
	for _, node := range cluster.Nodes {
		if node.Status == NodeStatus_NODE_CREATING {
			node.SetNodeStatus(NodeStatus_NODE_RUNNING)
		}
	}
	err = uc.clusterRuntime.MigrateToCluster(ctx, cluster)
	if err != nil {
		return err
	}
	return nil
}

package biz

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"

	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const (
	DefaultNodeGroupCpu        = 4
	DefaultNodeGroupMemory     = 8
	DefaultNodeGroupGpu        = 0
	DefaultNodeGroupMinSize    = 1
	DefaultNodeGroupMaxSize    = 5
	DefaultNodeGroupTargetSize = 3
	DefaultSSHPort             = 22

	ClusterLevel_BASIC_MaxSize    = 50
	ClusterLevel_ADVANCED_MaxSize = 100
	ClusterLevel_STANDARD_MaxSize = 200
)

type ClusterData interface {
	Save(context.Context, *Cluster) error
	Get(context.Context, int64) (*Cluster, error)
	GetByName(context.Context, string) (*Cluster, error)
	List(context.Context, *Cluster) ([]*Cluster, error)
	Delete(context.Context, int64) error
}

type ClusterInfrastructure interface {
	Start(context.Context, *Cluster) error
	Stop(context.Context, *Cluster) error
	GetRegions(context.Context, *Cluster) error
	MigrateToBostionHost(context.Context, *Cluster) error
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

type ClusterUsecase struct {
	clusterData           ClusterData
	clusterInfrastructure ClusterInfrastructure
	clusterRuntime        ClusterRuntime
	locks                 map[int64]*sync.Mutex
	locksMux              sync.Mutex
	eventChan             chan *Cluster
	conf                  *conf.Bootstrap
	log                   *log.Helper
}

func NewClusterUseCase(conf *conf.Bootstrap, clusterData ClusterData, clusterInfrastructure ClusterInfrastructure, clusterRuntime ClusterRuntime, logger log.Logger) *ClusterUsecase {
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

func (c *Cluster) SettingClusterAvailability() {
	maxNodeNumber := 0
	for _, nodeGroup := range c.NodeGroups {
		maxNodeNumber += int(nodeGroup.TargetSize)
	}
	if maxNodeNumber == 0 {
		return
	}
	if maxNodeNumber < ClusterLevel_BASIC_MaxSize {
		c.Level = ClusterLevel_BASIC
	}
	if maxNodeNumber < ClusterLevel_ADVANCED_MaxSize && maxNodeNumber >= ClusterLevel_BASIC_MaxSize {
		c.Level = ClusterLevel_ADVANCED
	}
	if maxNodeNumber >= ClusterLevel_ADVANCED_MaxSize {
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

func (c *Cluster) SettingCloudClusterInit() {
	nodegroup := &NodeGroup{Id: uuid.NewString(), ClusterId: c.Id}
	c.NodeGroups = append(c.NodeGroups, nodegroup)
	nodegroup.Type = NodeGroupType_NORMAL
	nodegroup.Cpu = DefaultNodeGroupCpu
	nodegroup.Memory = DefaultNodeGroupMemory
	nodegroup.Arch = NodeArchType_AMD64
	nodegroup.TargetSize = DefaultNodeGroupTargetSize
	nodegroup.MinSize = DefaultNodeGroupMinSize
	nodegroup.MaxSize = DefaultNodeGroupMaxSize
	nodegroup.Name = strings.Join([]string{c.Name, NodeGroupType_NORMAL.String()}, "-")
	if c.Type.IsIntegratedCloud() {
		return
	}
	labels := c.generateNodeLables(nodegroup)
	for i := 0; i < int(nodegroup.TargetSize); i++ {
		node := &Node{
			Name:        strings.Join([]string{nodegroup.Name, uuid.NewString()}, "-"),
			Status:      NodeStatus_NODE_CREATING,
			ClusterId:   c.Id,
			NodeGroupId: nodegroup.Id,
			Role:        NodeRole_WORKER,
			Labels:      labels,
		}
		if i == 0 {
			node.Role = NodeRole_MASTER
		}
		c.Nodes = append(c.Nodes, node)
	}
	c.BostionHost = &BostionHost{
		Id:        uuid.NewString(),
		ClusterId: c.Id,
		Hostname:  fmt.Sprintf("%s-bostion", c.Name),
		Status:    NodeStatus_NODE_CREATING,
		Cpu:       DefaultNodeGroupCpu,
		Memory:    DefaultNodeGroupMemory,
		SshPort:   DefaultSSHPort,
	}
	c.SecurityGroups = []*SecurityGroup{
		{
			Id:          uuid.NewString(),
			StartPort:   6443,
			EndPort:     6443,
			Protocol:    "TCP",
			IpCidr:      "0.0.0.0/0",
			ClusterId:   c.Id,
			Name:        fmt.Sprintf("%s-%s", c.Name, "apiserver"),
			Description: "apiserver sg",
		},
		{
			Id:          uuid.NewString(),
			StartPort:   10250,
			EndPort:     10255,
			Protocol:    "TCP",
			IpCidr:      "0.0.0.0/0",
			ClusterId:   c.Id,
			Name:        fmt.Sprintf("%s-%s", c.Name, "kubelet"),
			Description: "kubelet sg",
		},
		{
			Id:          uuid.NewString(),
			StartPort:   443,
			EndPort:     443,
			Protocol:    "TCP",
			IpCidr:      "0.0.0.0/0",
			ClusterId:   c.Id,
			Name:        fmt.Sprintf("%s-%s", c.Name, "ingresscontroller-https"),
			Description: "ingress controller https sg",
		},
		{
			Id:          uuid.NewString(),
			StartPort:   80,
			EndPort:     80,
			Protocol:    "TCP",
			IpCidr:      "0.0.0.0/0",
			ClusterId:   c.Id,
			Name:        fmt.Sprintf("%s-%s", c.Name, "ingresscontroller-http"),
			Description: "ingress controller http sg",
		},
		{
			Id:          uuid.NewString(),
			StartPort:   22,
			EndPort:     22,
			Protocol:    "TCP",
			IpCidr:      "0.0.0.0/0",
			ClusterId:   c.Id,
			Name:        fmt.Sprintf("%s-%s", c.Name, "infrastructure-ssh"),
			Description: "infrastructure remote shell sg",
		},
	}
}

func (c ClusterType) IsCloud() bool {
	return c != ClusterType_LOCAL && c != ClusterType_KUBERNETES
}

func (c ClusterType) IsIntegratedCloud() bool {
	return c == ClusterType_AWS_EKS || c == ClusterType_ALICLOUD_AKS
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
	err := uc.clusterInfrastructure.GetRegions(ctx, cluster)
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
		err = uc.clusterInfrastructure.UnInstall(ctx, cluster)
		if err != nil {
			return err
		}
		err = uc.clusterInfrastructure.Stop(ctx, cluster)
		if err != nil {
			return err
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
	err = uc.clusterRuntime.HandlerNodes(ctx, cluster)
	if err != nil {
		return err
	}
	err = uc.clusterInfrastructure.Start(ctx, cluster)
	if err != nil {
		return err
	}
	err = uc.clusterInfrastructure.HandlerNodes(ctx, cluster)
	if err != nil {
		return err
	}
	return
}

func (uc *ClusterUsecase) handlerClusterNotInstalled(ctx context.Context, cluster *Cluster) error {
	err := uc.clusterInfrastructure.GetRegions(ctx, cluster)
	if err != nil {
		return err
	}
	if cluster.Type.IsCloud() {
		cluster.SettingCloudClusterInit()
		cluster.SettingClusterAvailability()
	}
	err = uc.clusterInfrastructure.Start(ctx, cluster)
	if err != nil {
		return err
	}
	if uc.conf.Cluster.GetEnv() == conf.Env_EnvLocal {
		err = uc.clusterData.Save(ctx, cluster)
		if err != nil {
			return err
		}
		err = uc.clusterInfrastructure.MigrateToBostionHost(ctx, cluster)
		if err != nil {
			return err
		}
		return nil
	}
	err = uc.clusterInfrastructure.GetNodesSystemInfo(ctx, cluster)
	if err != nil {
		return err
	}
	err = uc.clusterInfrastructure.Install(ctx, cluster)
	if err != nil {
		return err
	}
	err = uc.clusterRuntime.MigrateToCluster(ctx, cluster)
	if err != nil {
		return err
	}
	return nil
}

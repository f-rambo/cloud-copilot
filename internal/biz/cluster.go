package biz

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"gorm.io/gorm"
)

const (
	ClusterPackageName         = "cluster"
	ClusterShellPackageName    = "shell"
	ClusterResroucePackageName = "resource"

	ClusterPoolNumber = 10
)

var ErrClusterNotFound error = errors.New("cluster not found")

type Cluster struct {
	ID                   int64                             `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name                 string                            `json:"name" gorm:"column:name; default:''; NOT NULL"`
	Version              string                            `json:"version" gorm:"column:version; default:''; NOT NULL"`
	ApiServerAddress     string                            `json:"api_server_address" gorm:"column:api_server_address; default:''; NOT NULL"`
	Config               string                            `json:"config" gorm:"column:config; default:''; NOT NULL;"`
	Addons               string                            `json:"addons" gorm:"column:addons; default:''; NOT NULL;"`
	AddonsConfig         string                            `json:"addons_config" gorm:"column:addons_config; default:''; NOT NULL;"`
	Status               ClusterStatus                     `json:"status" gorm:"column:status; default:0; NOT NULL;"`
	Type                 ClusterType                       `json:"type" gorm:"column:type; default:''; NOT NULL;"`
	Level                ClusterLevel                      `json:"level" gorm:"column:level; default:''; NOT NULL;"`
	PublicKey            string                            `json:"public_key" gorm:"column:public_key; default:''; NOT NULL;"`
	PrivateKey           string                            `json:"private_key" gorm:"column:private_key; default:''; NOT NULL;"`
	Connections          string                            `json:"connections" gorm:"column:connections; default:''; NOT NULL"`
	CertificateAuthority string                            `json:"certificate_authority" gorm:"column:certificate_authority; default:''; NOT NULL"`
	Region               string                            `json:"region" gorm:"column:region; default:''; NOT NULL;"`
	IpCidr               string                            `json:"ip_cidr" gorm:"column:ip_cidr; default:''; NOT NULL;"`
	AccessID             string                            `json:"access_id" gorm:"column:access_id; default:''; NOT NULL;"`
	AccessKey            string                            `json:"access_key" gorm:"column:access_key; default:''; NOT NULL;"`
	MasterIP             string                            `json:"master_ip" gorm:"column:master_ip; default:''; NOT NULL;"`
	MasterUser           string                            `json:"master_user" gorm:"column:master_user; default:''; NOT NULL;"`
	Token                string                            `json:"token" gorm:"column:token; default:''; NOT NULL;"`
	CAData               string                            `json:"ca_data" gorm:"column:ca_data; default:''; NOT NULL;"`
	CertData             string                            `json:"cert_data" gorm:"column:cert_data; default:''; NOT NULL;"`
	KeyData              string                            `json:"key_data" gorm:"column:key_data; default:''; NOT NULL;"`
	BostionHost          *BostionHost                      `json:"bostion_host" gorm:"-"`
	Nodes                []*Node                           `json:"nodes" gorm:"-"`
	NodeGroups           []*NodeGroup                      `json:"node_groups" gorm:"-"`
	CloudResources       map[ResourceType][]*CloudResource `json:"cloud_resources" gorm:"-"`
	CloudResourcesJson   string                            `json:"cloud_resources_json" gorm:"column:cloud_resources_json; default:''; NOT NULL;"`
	gorm.Model
}

type ClusterType string

const (
	ClusterTypeLocal       ClusterType = "local"
	ClusterTypeAWSEc2      ClusterType = "aws_ec2"
	ClusterTypeAWSEks      ClusterType = "aws_eks"
	ClusterTypeAliCloudEcs ClusterType = "alicloud_ecs"
	ClusterTypeAliCloudAks ClusterType = "alicloud_aks"
)

func (c ClusterType) String() string {
	return string(c)
}

func ClusterTypes() []ClusterType {
	return []ClusterType{
		ClusterTypeLocal,
		ClusterTypeAWSEc2,
		ClusterTypeAWSEks,
		ClusterTypeAliCloudEcs,
		ClusterTypeAliCloudAks,
	}
}

type ClusterLevel string

const (
	ClusterLevelBasic    ClusterLevel = "basic"
	ClusterLevelStandard ClusterLevel = "standard"
	ClusterLevelAdvanced ClusterLevel = "advanced"
)

func (c ClusterLevel) String() string {
	return string(c)
}

type ClusterStatus uint8
type ClusterStatusName string

const (
	ClusterStatusUnspecified ClusterStatus = 0
	ClusterStatusRunning     ClusterStatus = 1
	ClusterStatusDeleted     ClusterStatus = 2
	ClusterStatusStarting    ClusterStatus = 3
	ClusterStatusStopping    ClusterStatus = 4

	ClusterStatusNameUnspecified ClusterStatusName = "unspecified"
	ClusterStatusNameRunning     ClusterStatusName = "running"
	ClusterStatusNameDeleted     ClusterStatusName = "deleted"
	ClusterStatusNameStarting    ClusterStatusName = "starting"
	ClusterStatusNameStopping    ClusterStatusName = "stopping"
)

var ClusterStatusNameMap = map[ClusterStatus]ClusterStatusName{
	ClusterStatusUnspecified: ClusterStatusNameUnspecified,
	ClusterStatusRunning:     ClusterStatusNameRunning,
	ClusterStatusDeleted:     ClusterStatusNameDeleted,
	ClusterStatusStarting:    ClusterStatusNameStarting,
	ClusterStatusStopping:    ClusterStatusNameStopping,
}

func (s ClusterStatus) String() string {
	statusName, ok := ClusterStatusNameMap[s]
	if !ok {
		return string(ClusterStatusNameMap[ClusterStatusUnspecified])
	}
	return string(statusName)
}

func (c ClusterType) IsCloud() bool {
	return c != ClusterTypeLocal
}

func (c ClusterType) IsIntegratedCloud() bool {
	return c == ClusterTypeAWSEks || c == ClusterTypeAliCloudAks
}

func isClusterEmpty(c *Cluster) bool {
	if c == nil {
		return true
	}
	if c.ID == 0 {
		return true
	}
	return false
}

func (c *Cluster) IsDeleteed() bool {
	return c.DeletedAt.Valid
}

// ResourceType represents the type of cloud resource
type ResourceType string

const (
	ResourceTypeVPC               ResourceType = "VPC"
	ResourceTypeSubnet            ResourceType = "Subnet"
	ResourceTypeInternetGateway   ResourceType = "InternetGateway"
	ResourceTypeNATGateway        ResourceType = "NATGateway"
	ResourceTypeRouteTable        ResourceType = "RouteTable"
	ResourceTypeSecurityGroup     ResourceType = "SecurityGroup"
	ResourceTypeLoadBalancer      ResourceType = "LoadBalancer"
	ResourceTypeElasticIP         ResourceType = "ElasticIP"
	ResourceTypeAvailabilityZones ResourceType = "AvailabilityZones"
	ResourceTypeKeyPair           ResourceType = "KeyPair"
	ResourceTypeVpcEndpointS3     ResourceType = "VpcEndpointS3"
)

// CloudResource represents a cloud provider resource
type CloudResource struct {
	Name         string
	ID           string
	AssociatedID any // node id node group id cluster id
	Type         ResourceType
	Tags         map[string]string
	Value        any
	SubResources []*CloudResource // For resources that contain other resources
}

func (c *Cluster) GetCloudResource(resourceType ResourceType) []*CloudResource {
	resources, ok := c.CloudResources[resourceType]
	if !ok {
		return nil
	}
	return resources
}

func (c *Cluster) AddCloudResource(resourceType ResourceType, resource *CloudResource) {
	if c.CloudResources == nil {
		c.CloudResources = make(map[ResourceType][]*CloudResource)
	}
	if c.CloudResources[resourceType] == nil {
		c.CloudResources[resourceType] = []*CloudResource{}
	}
	resource.Type = resourceType
	c.CloudResources[resourceType] = append(c.CloudResources[resourceType], resource)
}

func (c *Cluster) AddSubCloudResource(resourceType ResourceType, parentID string, resource *CloudResource) {
	cloudResource := c.GetCloudResourceByID(resourceType, parentID)
	if cloudResource == nil {
		return
	}
	if cloudResource.SubResources == nil {
		cloudResource.SubResources = []*CloudResource{}
	}
	resource.Type = resourceType
	cloudResource.SubResources = append(cloudResource.SubResources, resource)
}

func (c *Cluster) GetCloudResourceByName(resourceType ResourceType, name string) *CloudResource {
	for _, resource := range c.CloudResources[resourceType] {
		if resource.Name == name {
			return resource
		}
	}
	return nil
}

func (c *Cluster) GetCloudResourceByID(resourceType ResourceType, id string) *CloudResource {
	if resources, ok := c.CloudResources[resourceType]; ok {
		resource := getCloudResourceByID(resources, id)
		if resource != nil {
			return resource
		}
	}
	return nil
}

func getCloudResourceByID(cloudResources []*CloudResource, id string) *CloudResource {
	for _, resource := range cloudResources {
		if resource.ID == id {
			return resource
		}
		if resource.SubResources != nil && len(resource.SubResources) > 0 {
			subResource := getCloudResourceByID(resource.SubResources, id)
			if subResource != nil {
				return subResource
			}
		}
	}
	return nil
}

func (c *Cluster) GetSingleCloudResource(resourceType ResourceType) *CloudResource {
	resources := c.GetCloudResource(resourceType)
	if len(resources) == 0 {
		return nil
	}
	return resources[0]
}

// getCloudResource by resourceType and tag value and tag key
func (c *Cluster) GetCloudResourceByTags(resourceType ResourceType, tagKeyValues ...string) []*CloudResource {
	if len(tagKeyValues)%2 != 0 {
		return nil
	}
	cloudResources := make([]*CloudResource, 0)
	for _, resource := range c.GetCloudResource(resourceType) {
		if resource.Tags == nil {
			continue
		}
		match := true
		for i := 0; i < len(tagKeyValues); i += 2 {
			tagKey := tagKeyValues[i]
			tagValue := tagKeyValues[i+1]
			if resource.Tags[tagKey] != tagValue {
				match = false
				break
			}
		}
		if match {
			cloudResources = append(cloudResources, resource)
		}
	}
	if len(cloudResources) == 0 {
		return nil
	}
	return cloudResources
}

// delete cloud resource by resourceType
func (c *Cluster) DeleteCloudResource(resourceType ResourceType) {
	if c.CloudResources == nil {
		return
	}
	if c.CloudResources[resourceType] == nil {
		return
	}
	c.CloudResources[resourceType] = []*CloudResource{}
}

// delete cloud resource by resourceType and id
func (c *Cluster) DeleteCloudResourceByID(resourceType ResourceType, id string) {
	if c.CloudResources == nil {
		return
	}
	if c.CloudResources[resourceType] == nil {
		return
	}
	for i, resource := range c.CloudResources[resourceType] {
		if resource.ID == id {
			c.CloudResources[resourceType] = append(c.CloudResources[resourceType][:i], c.CloudResources[resourceType][i+1:]...)
			break
		}
	}
}

// delete cloud resource by resourceType and tag value and tag key
func (c *Cluster) DeleteCloudResourceByTags(resourceType ResourceType, tagKeyValues ...string) {
	if c.CloudResources == nil {
		return
	}
	if c.CloudResources[resourceType] == nil {
		return
	}
	for i, resource := range c.CloudResources[resourceType] {
		if resource.Tags == nil {
			continue
		}
		match := true
		for j := 0; j < len(tagKeyValues); j += 2 {
			tagKey := tagKeyValues[j]
			tagValue := tagKeyValues[j+1]
			if resource.Tags[tagKey] != tagValue {
				match = false
				break
			}
		}
		if match {
			c.CloudResources[resourceType] = append(c.CloudResources[resourceType][:i], c.CloudResources[resourceType][i+1:]...)
		}
	}
}

func (c *Cluster) SettingSpecifications() {
	if !c.Type.IsCloud() {
		return
	}
	if len(c.NodeGroups) != 0 || len(c.Nodes) != 0 {
		return
	}
	ipCidr := os.Getenv("CLUSTER_IP_CIDR")
	if ipCidr == "" {
		ipCidr = "10.0.0.0/16"
	}
	c.IpCidr = ipCidr
	nodegroup := c.NewNodeGroup()
	nodegroup.Type = NodeGroupTypeNormal
	c.GenerateNodeGroupName(nodegroup)
	nodegroup.CPU = 2
	nodegroup.Memory = 4
	nodegroup.TargetSize = 3
	nodegroup.MinSize = 1
	nodegroup.MaxSize = 5
	if c.Level != ClusterLevelBasic {
		nodegroup.CPU = 4
		nodegroup.Memory = 8
		nodegroup.TargetSize = 5
		nodegroup.MinSize = 5
		nodegroup.MaxSize = 10
	}
	if nodegroup.TargetSize > 0 {
		c.NodeGroups = append(c.NodeGroups, nodegroup)
	}
	if c.Type.IsIntegratedCloud() {
		return
	}
	labels := c.generateNodeLables(nodegroup)
	for i := 0; i < int(nodegroup.MinSize); i++ {
		node := &Node{
			Name:        fmt.Sprintf("%s-node-%s-%s", c.Name, nodegroup.Name, utils.GetRandomString()),
			Status:      NodeStatusCreating,
			ClusterID:   c.ID,
			NodeGroupID: nodegroup.ID,
		}
		if i < 3 {
			node.Role = NodeRoleMaster
		} else {
			node.Role = NodeRoleWorker
		}
		node.Labels = labels
		c.Nodes = append(c.Nodes, node)
	}
	c.BostionHost = &BostionHost{
		ID:        1,
		ClusterID: c.ID,
		Hostname:  "bostion-host",
		Status:    NodeStatusCreating,
		CPU:       2,
		Memory:    4,
		SshPort:   22,
	}
}

type NodeGroup struct {
	ID               string        `json:"id" gorm:"column:id;primaryKey; NOT NULL"`
	Name             string        `json:"name" gorm:"column:name; default:''; NOT NULL"`
	Type             NodeGroupType `json:"type" gorm:"column:type; default:''; NOT NULL;"`
	Image            string        `json:"image" gorm:"column:image; default:''; NOT NULL"`
	ImageDescription string        `json:"image_description" gorm:"column:image_description; default:''; NOT NULL"`
	OS               string        `json:"os" gorm:"column:os; default:''; NOT NULL"`
	ARCH             string        `json:"arch" gorm:"column:arch; default:''; NOT NULL"`
	CPU              int32         `json:"cpu" gorm:"column:cpu; default:0; NOT NULL"`
	Memory           int32         `json:"memory" gorm:"column:memory; default:0; NOT NULL"`
	GPU              int32         `json:"gpu" gorm:"column:gpu; default:0; NOT NULL"`
	GpuSpec          string        `json:"gpu_spec" gorm:"column:gpu_spec; default:''; NOT NULL"`
	DataDisk         int32         `json:"data_disk" gorm:"column:data_disk; default:0; NOT NULL"`
	RootDeviceName   string        `json:"root_device_name" gorm:"column:root_device_name; default:''; NOT NULL"`
	DataDeviceName   string        `json:"data_device_name" gorm:"column:data_device_name; default:''; NOT NULL"`
	MinSize          int32         `json:"min_size" gorm:"column:min_size; default:0; NOT NULL"`
	MaxSize          int32         `json:"max_size" gorm:"column:max_size; default:0; NOT NULL"`
	TargetSize       int32         `json:"target_size" gorm:"column:target_size; default:0; NOT NULL"`
	InstanceType     string        `json:"instance_type" gorm:"column:instance_type; default:''; NOT NULL"`
	DefaultUsername  string        `json:"default_username" gorm:"column:default_username; default:''; NOT NULL"`
	NodePrice        float64       `json:"node_price" gorm:"column:node_price; default:0; NOT NULL;"`
	PodPrice         float64       `json:"pod_price" gorm:"column:pod_price; default:0; NOT NULL;"`
	Zone             string        `json:"zone" gorm:"column:zone; default:''; NOT NULL"`
	SubnetIpCidr     string        `json:"subnet_ip_cidr" gorm:"column:subnet_ip_cidr; default:''; NOT NULL"`
	NodeInitScript   string        `json:"cloud_init_script" gorm:"column:cloud_init_script; default:''; NOT NULL"`
	ClusterID        int64         `json:"cluster_id" gorm:"column:cluster_id; default:0; NOT NULL"`
}

type NodeGroupType string

func (n NodeGroupType) String() string {
	return string(n)
}

const (
	NodeGroupTypeNormal          NodeGroupType = "normal"
	NodeGroupTypeHighComputation NodeGroupType = "highComputation"
	NodeGroupTypeGPUAcceleraterd NodeGroupType = "gpuAcceleraterd"
	NodeGroupTypeHighMemory      NodeGroupType = "highMemory"
	NodeGroupTypeLargeHardDisk   NodeGroupType = "largeHardDisk"
)

func (c *Cluster) NewNodeGroup() *NodeGroup {
	return &NodeGroup{
		ID:        uuid.New().String(),
		ClusterID: c.ID,
	}
}

func (ng *NodeGroup) SetTargetSize(size int32) {
	ng.TargetSize = size
}

type NodeGroups []*NodeGroup

func (n NodeGroups) Len() int {
	return len(n)
}

func (n NodeGroups) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

func (n NodeGroups) Less(i, j int) bool {
	if n[i] == nil || n[j] == nil {
		return false
	}
	if n[i].Memory == n[j].Memory {
		return n[i].CPU < n[j].CPU
	}
	return n[i].Memory < n[j].Memory
}

func (c *Cluster) GetNodeGroup(nodeGroupId string) *NodeGroup {
	for _, nodeGroup := range c.NodeGroups {
		if nodeGroup.ID == nodeGroupId {
			return nodeGroup
		}
	}
	return nil
}

func (c *Cluster) GetNodeGroupByName(nodeGroupName string) *NodeGroup {
	for _, nodeGroup := range c.NodeGroups {
		if nodeGroup.Name == nodeGroupName {
			return nodeGroup
		}
	}
	return nil
}

func (c *Cluster) GenerateNodeGroupName(nodeGroup *NodeGroup) {
	nodeGroup.Name = strings.Join([]string{
		c.Name,
		nodeGroup.Type.String(),
		nodeGroup.OS,
		nodeGroup.ARCH,
		cast.ToString(nodeGroup.CPU),
		cast.ToString(nodeGroup.Memory),
		cast.ToString(nodeGroup.GPU),
		cast.ToString(nodeGroup.GpuSpec),
	}, "-")
}

type Node struct {
	ID          int64      `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name        string     `json:"name" gorm:"column:name; default:''; NOT NULL"`
	Labels      string     `json:"labels" gorm:"column:labels; default:''; NOT NULL"`
	InternalIP  string     `json:"internal_ip" gorm:"column:internal_ip; default:''; NOT NULL"`
	ExternalIP  string     `json:"external_ip" gorm:"column:external_ip; default:''; NOT NULL"`
	User        string     `json:"user" gorm:"column:user; default:''; NOT NULL"`
	Role        NodeRole   `json:"role" gorm:"column:role; default:''; NOT NULL;"`
	Status      NodeStatus `json:"status" gorm:"column:status; default:0; NOT NULL;"`
	ClusterID   int64      `json:"cluster_id" gorm:"column:cluster_id; default:0; NOT NULL"`
	NodeGroupID string     `json:"node_group_id" gorm:"column:node_group_id; default:''; NOT NULL"`
	InstanceID  string     `json:"instance_id" gorm:"column:instance_id; default:''; NOT NULL"`
	ErrorInfo   string     `json:"error_info" gorm:"column:error_info; default:''; NOT NULL"`
	gorm.Model
}

type NodeRole string

const (
	NodeRoleMaster NodeRole = "master"
	NodeRoleWorker NodeRole = "worker"
	NodeRoleEdge   NodeRole = "edge"
)

func (n NodeRole) String() string {
	return string(n)
}

type NodeStatus uint8
type NodeStatusName string

const (
	NodeStatusUnspecified NodeStatus = 0
	NodeStatusRunning     NodeStatus = 1
	NodeStatusCreating    NodeStatus = 2
	NodeStatusDeleting    NodeStatus = 3
	NodeStatusDeleted     NodeStatus = 4
	NodeStatusError       NodeStatus = 5

	NodeStatusUnspecifiedName NodeStatusName = "unspecified"
	NodeStatusRunningName     NodeStatusName = "running"
	NodeStatusCreatingName    NodeStatusName = "creating"
	NodeStatusDeletingName    NodeStatusName = "deleting"
	NodeStatusDeletedName     NodeStatusName = "deleted"
	NodeStatusErrorName       NodeStatusName = "error"
)

var NodeStatusNameMap = map[NodeStatus]NodeStatusName{
	NodeStatusUnspecified: NodeStatusUnspecifiedName,
	NodeStatusRunning:     NodeStatusRunningName,
	NodeStatusCreating:    NodeStatusCreatingName,
	NodeStatusDeleting:    NodeStatusDeletingName,
	NodeStatusDeleted:     NodeStatusDeletedName,
	NodeStatusError:       NodeStatusErrorName,
}

func (s NodeStatus) String() string {
	statusName, ok := NodeStatusNameMap[s]
	if !ok {
		return string(NodeStatusNameMap[NodeStatusUnspecified])
	}
	return string(statusName)
}

type BostionHost struct {
	ID               int64      `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	User             string     `json:"user" gorm:"column:user; default:''; NOT NULL"`
	Image            string     `json:"image" gorm:"column:image; default:''; NOT NULL"`
	ImageDescription string     `json:"image_description" gorm:"column:image_description; default:''; NOT NULL"`
	OS               string     `json:"os" gorm:"column:os; default:''; NOT NULL"`
	ARCH             string     `json:"arch" gorm:"column:arch; default:''; NOT NULL"`
	CPU              int32      `json:"cpu" gorm:"column:cpu; default:0; NOT NULL"`
	Memory           int32      `json:"memory" gorm:"column:memory; default:0; NOT NULL"`
	Hostname         string     `json:"hostname" gorm:"column:hostname; default:''; NOT NULL"`
	ExternalIP       string     `json:"external_ip" gorm:"column:external_ip; default:''; NOT NULL"`
	InternalIP       string     `json:"internal_ip" gorm:"column:internal_ip; default:''; NOT NULL"`
	SshPort          int32      `json:"ssh_port" gorm:"column:ssh_port; default:0; NOT NULL"`
	Status           NodeStatus `json:"status" gorm:"column:status; default:0; NOT NULL;"`
	InstanceID       string     `json:"instance_id" gorm:"column:instance_id; default:''; NOT NULL"`
	ClusterID        int64      `json:"cluster_id" gorm:"column:cluster_id; default:0; NOT NULL"`
	gorm.Model
}

type ClusterRepo interface {
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
	clusterRepo           ClusterRepo
	clusterInfrastructure ClusterInfrastructure
	clusterRuntime        ClusterRuntime
	locks                 map[int64]*sync.Mutex
	locksMux              sync.Mutex
	eventChan             chan *Cluster
	conf                  *conf.Bootstrap
	log                   *log.Helper
}

func NewClusterUseCase(conf *conf.Bootstrap, clusterRepo ClusterRepo, clusterInfrastructure ClusterInfrastructure, clusterRuntime ClusterRuntime, logger log.Logger) *ClusterUsecase {
	c := &ClusterUsecase{
		clusterRepo:           clusterRepo,
		clusterInfrastructure: clusterInfrastructure,
		clusterRuntime:        clusterRuntime,
		conf:                  conf,
		log:                   log.NewHelper(logger),
		locks:                 make(map[int64]*sync.Mutex),
		eventChan:             make(chan *Cluster, ClusterPoolNumber),
	}
	go c.clusterRunner()
	return c
}

func (uc *ClusterUsecase) Get(ctx context.Context, id int64) (*Cluster, error) {
	cluster, err := uc.clusterRepo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (uc *ClusterUsecase) List(ctx context.Context) ([]*Cluster, error) {
	return uc.clusterRepo.List(ctx, nil)
}

func (uc *ClusterUsecase) Delete(ctx context.Context, clusterID int64) error {
	cluster, err := uc.clusterRepo.Get(ctx, clusterID)
	if err != nil {
		return err
	}
	if isClusterEmpty(cluster) {
		return nil
	}
	for _, node := range cluster.Nodes {
		node.Status = NodeStatusDeleting
	}
	err = uc.clusterRepo.Save(ctx, cluster)
	if err != nil {
		return err
	}
	err = uc.clusterRepo.Delete(ctx, clusterID)
	if err != nil {
		return err
	}
	return nil
}

func (uc *ClusterUsecase) Save(ctx context.Context, cluster *Cluster) error {
	data, err := uc.clusterRepo.GetByName(ctx, cluster.Name)
	if err != nil {
		return err
	}
	if !isClusterEmpty(data) && isClusterEmpty(cluster) {
		return errors.New("cluster name already exists")
	}
	if cluster.Level == "" {
		cluster.Level = ClusterLevelBasic
	}
	err = uc.clusterRepo.Save(ctx, cluster)
	if err != nil {
		return err
	}
	uc.apply(cluster)
	return nil
}

func (uc *ClusterUsecase) GetRegions(ctx context.Context, cluster *Cluster) ([]string, error) {
	if cluster.Type == ClusterTypeLocal {
		return []string{}, nil
	}
	err := uc.clusterInfrastructure.GetRegions(ctx, cluster)
	if err != nil {
		return nil, err
	}
	regionNames := make([]string, 0)
	for _, region := range cluster.GetCloudResource(ResourceTypeAvailabilityZones) {
		regionNames = append(regionNames, region.Name)
	}
	return regionNames, nil
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

func (uc *ClusterUsecase) apply(cluster *Cluster) {
	uc.eventChan <- cluster
}

func (uc *ClusterUsecase) clusterRunner() {
	for event := range uc.eventChan {
		go uc.handleEvent(context.TODO(), event)
	}
}

func (uc *ClusterUsecase) handleEvent(ctx context.Context, cluster *Cluster) (err error) {
	lock := uc.getLock(cluster.ID)
	lock.Lock()
	defer func() {
		if err != nil {
			return
		}
		lock.Unlock()
		err = uc.clusterRepo.Save(ctx, cluster)
	}()
	if cluster.IsDeleteed() {
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
	cluster.SettingSpecifications()
	err := uc.clusterInfrastructure.GetRegions(ctx, cluster)
	if err != nil {
		return err
	}
	if cluster.Level == ClusterLevelBasic {
		singleZone := cluster.GetSingleCloudResource(ResourceTypeAvailabilityZones)
		cluster.DeleteCloudResource(ResourceTypeAvailabilityZones)
		cluster.AddCloudResource(ResourceTypeAvailabilityZones, singleZone)
	}
	err = uc.clusterInfrastructure.Start(ctx, cluster)
	if err != nil {
		return err
	}
	if uc.conf.Server.GetEnv() == conf.EnvLocal {
		err = uc.clusterRepo.Save(ctx, cluster)
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

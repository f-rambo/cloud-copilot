package biz

import (
	"context"
	"fmt"
	"strings"

	"github.com/f-rambo/ocean/internal/conf"
	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const (
	ClusterPackageName = "cluster"
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
	KubeConfig           string                            `json:"kube_config" gorm:"column:kube_config; default:''; NOT NULL; type:json"`
	PublicKey            string                            `json:"public_key" gorm:"column:public_key; default:''; NOT NULL;"`
	PrivateKey           string                            `json:"private_key" gorm:"column:private_key; default:''; NOT NULL;"`
	Connections          string                            `json:"connections" gorm:"column:connections; default:''; NOT NULL"`
	CertificateAuthority string                            `json:"certificate_authority" gorm:"column:certificate_authority; default:''; NOT NULL"`
	Region               string                            `json:"region" gorm:"column:region; default:''; NOT NULL;"`
	IpCidr               string                            `json:"ip_cidr" gorm:"column:ip_cidr; default:''; NOT NULL;"`
	AccessID             string                            `json:"access_id" gorm:"column:access_id; default:''; NOT NULL;"`
	AccessKey            string                            `json:"access_key" gorm:"column:access_key; default:''; NOT NULL;"`
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

type ClusterStatus uint8

func (s ClusterStatus) Uint8() uint8 {
	return uint8(s)
}

const (
	ClusterStatusUnspecified ClusterStatus = 0
	ClusterStatusRunning     ClusterStatus = 1
	ClusterStatusDeleted     ClusterStatus = 2
	ClusterStatusStarting    ClusterStatus = 3
	ClusterStatusStopping    ClusterStatus = 4
)

var (
	ClusterStatusName = map[uint8]string{
		0: "unspecified",
		1: "running",
		2: "deleted",
		3: "starting",
		4: "stopping",
	}
	ClusterStatusValue = map[string]uint8{
		"unspecified": 0,
		"running":     1,
		"deleted":     2,
		"starting":    3,
		"stopping":    4,
	}
)

func (s ClusterStatus) String() string {
	statusName, ok := ClusterStatusName[s.Uint8()]
	if !ok {
		return ClusterStatusName[0]
	}
	return statusName
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
	ResourceTypeVPC             ResourceType = "VPC"
	ResourceTypeSubnet          ResourceType = "Subnet"
	ResourceTypeInternetGateway ResourceType = "InternetGateway"
	ResourceTypeNATGateway      ResourceType = "NATGateway"
	ResourceTypeRouteTable      ResourceType = "RouteTable"
	ResourceTypeSecurityGroup   ResourceType = "SecurityGroup"
	ResourceTypeLoadBalancer    ResourceType = "LoadBalancer"
)

// CloudResource represents a cloud provider resource
type CloudResource struct {
	Name         string
	ID           string
	AssociatedID any // node id node group id cluster id
	Type         ResourceType
	Tags         map[string]string
	Value        string
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

func (c *Cluster) GetCloudResourceByName(resourceType ResourceType, name string) *CloudResource {
	for _, resource := range c.CloudResources[resourceType] {
		if resource.Name == name {
			return resource
		}
	}
	return nil
}

func (c *Cluster) GetCloudResourceByID(resourceType ResourceType, id string) *CloudResource {
	for _, resource := range c.CloudResources[resourceType] {
		if resource.ID == id {
			return resource
		}
	}
	return nil
}

func (c *Cluster) GetFirstCloudResource(resourceType ResourceType) *CloudResource {
	resources := c.GetCloudResource(resourceType)
	if len(resources) == 0 {
		return nil
	}
	return resources[0]
}

type NodeGroup struct {
	ID             string        `json:"id" gorm:"column:id;primaryKey; NOT NULL"`
	Name           string        `json:"name" gorm:"column:name; default:''; NOT NULL"`
	Type           NodeGroupType `json:"type" gorm:"column:type; default:''; NOT NULL;"`
	Image          string        `json:"image" gorm:"column:image; default:''; NOT NULL"`
	OS             string        `json:"os" gorm:"column:os; default:''; NOT NULL"`
	ARCH           string        `json:"arch" gorm:"column:arch; default:''; NOT NULL"`
	CPU            int32         `json:"cpu" gorm:"column:cpu; default:0; NOT NULL"`
	Memory         float64       `json:"memory" gorm:"column:memory; default:0; NOT NULL"`
	GPU            int32         `json:"gpu" gorm:"column:gpu; default:0; NOT NULL"`
	NodeInitScript string        `json:"cloud_init_script" gorm:"column:cloud_init_script; default:''; NOT NULL"`
	MinSize        int32         `json:"min_size" gorm:"column:min_size; default:0; NOT NULL"`
	MaxSize        int32         `json:"max_size" gorm:"column:max_size; default:0; NOT NULL"`
	TargetSize     int32         `json:"target_size" gorm:"column:target_size; default:0; NOT NULL"`
	SystemDisk     int32         `json:"system_disk" gorm:"column:system_disk; default:0; NOT NULL"`
	DataDisk       int32         `json:"data_disk" gorm:"column:data_disk; default:0; NOT NULL"`
	ClusterID      int64         `json:"cluster_id" gorm:"column:cluster_id; default:0; NOT NULL"`
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

func (c *Cluster) GenerateNodeGroupName(nodeGroup *NodeGroup) {
	nodeGroup.Name = strings.Join([]string{c.Name, nodeGroup.Type.String()}, "-")
}

type Node struct {
	ID                      int64      `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name                    string     `json:"name" gorm:"column:name; default:''; NOT NULL"`
	Labels                  string     `json:"labels" gorm:"column:labels; default:''; NOT NULL"`
	Kernel                  string     `json:"kernel" gorm:"column:kernel; default:''; NOT NULL"`
	ContainerRuntime        string     `json:"container_runtime" gorm:"column:container_runtime; default:''; NOT NULL"`
	Kubelet                 string     `json:"kubelet" gorm:"column:kubelet; default:''; NOT NULL"`
	KubeProxy               string     `json:"kube_proxy" gorm:"column:kube_proxy; default:''; NOT NULL"`
	SshPort                 int32      `json:"ssh_port" gorm:"column:ssh_port; default:0; NOT NULL"`
	GrpcPort                int32      `json:"grpc_port" gorm:"column:grpc_port; default:0; NOT NULL"`
	InternalIP              string     `json:"internal_ip" gorm:"column:internal_ip; default:''; NOT NULL"`
	ExternalIP              string     `json:"external_ip" gorm:"column:external_ip; default:''; NOT NULL"`
	User                    string     `json:"user" gorm:"column:user; default:''; NOT NULL"`
	Role                    NodeRole   `json:"role" gorm:"column:role; default:''; NOT NULL;"`
	Status                  NodeStatus `json:"status" gorm:"column:status; default:0; NOT NULL;"`
	ErrorInfo               string     `json:"error_info" gorm:"column:error_info; default:''; NOT NULL"`
	Zone                    string     `json:"zone" gorm:"column:zone; default:''; NOT NULL"`
	IpCidr                  string     `json:"ip_cidr" gorm:"column:ip_cidr; default:''; NOT NULL"`
	GpuSpec                 string     `json:"gpu_spec" gorm:"column:gpu_spec; default:''; NOT NULL"`
	SystemDisk              int32      `json:"system_disk" gorm:"column:system_disk; default:0; NOT NULL"`
	DataDisk                int32      `json:"data_disk" gorm:"column:data_disk; default:0; NOT NULL"`
	NodePrice               float64    `json:"node_price" gorm:"column:node_price; default:0; NOT NULL;"`
	PodPrice                float64    `json:"pod_price" gorm:"column:pod_price; default:0; NOT NULL;"`
	InternetMaxBandwidthOut int32      `json:"internet_max_bandwidth_out" gorm:"column:internet_max_bandwidth_out; default:0; NOT NULL"`
	ClusterID               int64      `json:"cluster_id" gorm:"column:cluster_id; default:0; NOT NULL"`
	NodeGroupID             string     `json:"node_group_id" gorm:"column:node_group_id; default:''; NOT NULL"`
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

const (
	NodeStatusUnspecified NodeStatus = 0
	NodeStatusRunning     NodeStatus = 1
	NodeStatusCreating    NodeStatus = 2
	NodeStatusDeleting    NodeStatus = 3
)

func (s NodeStatus) Uint8() uint8 {
	return uint8(s)
}

var (
	NodeStatusName = map[uint8]string{
		0: "unspecified",
		1: "instanceRunning",
		2: "instanceCreating",
		3: "instanceDeleting",
	}
	NodeStatusValue = map[string]uint8{
		"unspecified":      0,
		"instanceRunning":  1,
		"instanceCreating": 2,
		"instanceDeleting": 3,
	}
)

func (s NodeStatus) String() string {
	statusName, ok := NodeStatusName[s.Uint8()]
	if !ok {
		return NodeStatusName[0]
	}
	return statusName
}

type BostionHost struct {
	ID         int64  `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	User       string `json:"user" gorm:"column:user; default:''; NOT NULL"`
	ImageID    string `json:"image_id" gorm:"column:image_id; default:''; NOT NULL"`
	Image      string `json:"image" gorm:"column:image; default:''; NOT NULL"`
	OS         string `json:"os" gorm:"column:os; default:''; NOT NULL"`
	ARCH       string `json:"arch" gorm:"column:arch; default:''; NOT NULL"`
	Hostname   string `json:"hostname" gorm:"column:hostname; default:''; NOT NULL"`
	ExternalIP string `json:"external_ip" gorm:"column:external_ip; default:''; NOT NULL"`
	InternalIP string `json:"internal_ip" gorm:"column:internal_ip; default:''; NOT NULL"`
	SshPort    int32  `json:"ssh_port" gorm:"column:ssh_port; default:0; NOT NULL"`
	ClusterID  int64  `json:"cluster_id" gorm:"column:cluster_id; default:0; NOT NULL"`
	gorm.Model
}

type ClusterRepo interface {
	Save(context.Context, *Cluster) error
	Get(context.Context, int64) (*Cluster, error)
	GetByName(context.Context, string) (*Cluster, error)
	List(context.Context, *Cluster) ([]*Cluster, error)
	Delete(context.Context, int64) error
	Put(ctx context.Context, cluster *Cluster) error
	Watch(ctx context.Context) (*Cluster, error)
}

type ClusterInfrastructure interface {
	Start(context.Context, *Cluster) error
	Stop(context.Context, *Cluster) error
	GetRegions(context.Context, *Cluster) ([]string, error)
	MigrateToBostionHost(context.Context, *Cluster) error
	DistributeDaemonApp(context.Context, *Cluster) error
	GetNodesSystemInfo(context.Context, *Cluster) error
	Install(context.Context, *Cluster) error
	UnInstall(context.Context, *Cluster) error
	AddNodes(context.Context, *Cluster, []*Node) error
	RemoveNodes(context.Context, *Cluster, []*Node) error
}

type ClusterRuntime interface {
	CurrentCluster(context.Context, *Cluster) error
}

type ClusterUsecase struct {
	clusterRepo           ClusterRepo
	clusterInfrastructure ClusterInfrastructure
	clusterRuntime        ClusterRuntime
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
	}
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
	for _, node := range cluster.Nodes {
		if node.Name == "" {
			node.Name = fmt.Sprintf("%s-%s", cluster.Name, utils.GetRandomString())
		}
	}
	err = uc.clusterRepo.Save(ctx, cluster)
	if err != nil {
		return err
	}
	return nil
}

func (uc *ClusterUsecase) GetRegions(ctx context.Context, cluster *Cluster) ([]string, error) {
	if cluster.Type == ClusterTypeLocal {
		return []string{}, nil
	}
	return uc.clusterInfrastructure.GetRegions(ctx, cluster)
}

func (uc *ClusterUsecase) Apply(ctx context.Context, cluster *Cluster) error {
	return uc.clusterRepo.Put(ctx, cluster)
}

func (uc *ClusterUsecase) Watch(ctx context.Context) (*Cluster, error) {
	return uc.clusterRepo.Watch(ctx)
}

func (uc *ClusterUsecase) Reconcile(ctx context.Context, cluster *Cluster) (err error) {
	defer func() {
		if err != nil {
			return
		}
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
	err = uc.clusterInfrastructure.Start(ctx, cluster)
	if err != nil {
		return err
	}
	err = uc.handlerAddNode(ctx, cluster)
	if err != nil {
		return err
	}
	err = uc.handlerRemoveNode(ctx, cluster)
	if err != nil {
		return err
	}
	return
}

func (uc *ClusterUsecase) handlerClusterNotInstalled(ctx context.Context, cluster *Cluster) error {
	uc.settingSpecifications(cluster)
	err := uc.clusterInfrastructure.Start(ctx, cluster)
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
	err = uc.clusterInfrastructure.DistributeDaemonApp(ctx, cluster)
	if err != nil {
		return err
	}
	err = uc.clusterInfrastructure.GetNodesSystemInfo(ctx, cluster)
	if err != nil {
		return err
	}
	err = uc.clusterInfrastructure.Install(ctx, cluster)
	if err != nil {
		return err
	}
	return nil
}

func (uc *ClusterUsecase) handlerAddNode(ctx context.Context, cluster *Cluster) error {
	addNodes := make([]*Node, 0)
	for _, node := range cluster.Nodes {
		if node.Status == NodeStatusCreating {
			addNodes = append(addNodes, node)
		}
	}
	err := uc.clusterInfrastructure.AddNodes(ctx, cluster, addNodes)
	if err != nil {
		return err
	}
	for _, node := range cluster.Nodes {
		for _, n := range addNodes {
			if node.Name == n.Name {
				node.Status = NodeStatusRunning
			}
		}
	}
	return nil
}

func (uc *ClusterUsecase) handlerRemoveNode(ctx context.Context, cluster *Cluster) error {
	removeNodes := make([]*Node, 0)
	for _, node := range cluster.Nodes {
		if node.Status == NodeStatusDeleting {
			removeNodes = append(removeNodes, node)
		}
	}
	err := uc.clusterInfrastructure.RemoveNodes(ctx, cluster, removeNodes)
	if err != nil {
		return err
	}
	newNodes := make([]*Node, 0)
	for _, node := range cluster.Nodes {
		ok := false
		for _, n := range removeNodes {
			if node.Name == n.Name {
				ok = true
			}
		}
		if !ok {
			newNodes = append(newNodes, node)
		}
	}
	cluster.Nodes = newNodes
	return nil
}

// Setting specifications
func (uc *ClusterUsecase) settingSpecifications(cluster *Cluster) {
	if !cluster.Type.IsCloud() {
		return
	}
	if len(cluster.NodeGroups) != 0 || len(cluster.Nodes) != 0 {
		return
	}
	cluster.IpCidr = "10.0.0.0/16"
	nodegroup := cluster.NewNodeGroup()
	nodegroup.Type = NodeGroupTypeNormal
	cluster.GenerateNodeGroupName(nodegroup)
	nodegroup.CPU = 4
	nodegroup.Memory = 8
	nodegroup.TargetSize = 5
	nodegroup.MinSize = 1
	nodegroup.MaxSize = 10
	cluster.NodeGroups = append(cluster.NodeGroups, nodegroup)
	if cluster.Type.IsIntegratedCloud() {
		return
	}
	for i := 0; i < int(nodegroup.TargetSize); i++ {
		node := &Node{
			Name:        fmt.Sprintf("%s-%s-%s", cluster.Name, nodegroup.Name, utils.GetRandomString()),
			Status:      NodeStatusUnspecified,
			ClusterID:   cluster.ID,
			NodeGroupID: nodegroup.ID,
		}
		if i < 3 {
			node.Role = NodeRoleMaster
		} else {
			node.Role = NodeRoleWorker
		}
		node.Labels = cluster.generateNodeLables(nodegroup)
		cluster.Nodes = append(cluster.Nodes, node)
	}
}

package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"

	confPkg "github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

const (
	ClusterPoolNumber = 10

	ClusterKey ContextKey = "cluster"
)

var ErrClusterNotFound error = errors.New("cluster not found")

type EventSource int32

const (
	EventSource_UNSPECIFIED EventSource = 0
	EventSource_CLUSTER     EventSource = 1
	EventSource_APP         EventSource = 2
	EventSource_PROJECT     EventSource = 3
	EventSource_SERVICE     EventSource = 4
	EventSource_USER        EventSource = 5
)

type EventAction int32

const (
	EventAction_UNSPECIFIED EventAction = 0
	EventAction_CREATE      EventAction = 1
	EventAction_UPDATE      EventAction = 2
	EventAction_DELETE      EventAction = 3
)

type EventStatus int32

const (
	EventStatus_UNSPECIFIED EventStatus = 0
	EventStatus_PENDING     EventStatus = 1
	EventStatus_PROCESSING  EventStatus = 2
	EventStatus_SUCCESS     EventStatus = 3
	EventStatus_FAILED      EventStatus = 4
)

type Event struct {
	Id        string      `json:"id,omitempty" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name      string      `json:"name,omitempty" gorm:"column:name;default:'';NOT NULL"`
	Source    EventSource `json:"source,omitempty" gorm:"column:source;default:0;NOT NULL"`
	Action    EventAction `json:"action,omitempty" gorm:"column:action;default:0;NOT NULL"`
	Status    EventStatus `json:"status,omitempty" gorm:"column:status;default:0;NOT NULL"`
	SourceId  int64       `json:"source_id,omitempty" gorm:"column:source_id;default:0;NOT NULL"`
	Data      string      `json:"data,omitempty" gorm:"column:data;default:'';NOT NULL"`
	Error     string      `json:"error,omitempty" gorm:"column:error;default:'';NOT NULL"`
	CreatedAt string      `json:"created_at,omitempty" gorm:"column:created_at;default:'';NOT NULL"`
}

type ClusterNamespace int32

const (
	ClusterNamespace_cloudcopilot ClusterNamespace = 0
	ClusterNamespace_networking   ClusterNamespace = 1
	ClusterNamespace_storage      ClusterNamespace = 2
	ClusterNamespace_monitoring   ClusterNamespace = 3
	ClusterNamespace_toolkit      ClusterNamespace = 4
)

// ClusterNamespace to string
func (cn ClusterNamespace) String() string {
	switch cn {
	case ClusterNamespace_cloudcopilot:
		return "cloudcopilot"
	case ClusterNamespace_networking:
		return "networking"
	case ClusterNamespace_storage:
		return "storage"
	case ClusterNamespace_monitoring:
		return "monitoring"
	case ClusterNamespace_toolkit:
		return "toolkit"
	default:
		return "default"
	}
}

type ClusterProvider int32

const (
	ClusterProvider_UNSPECIFIED ClusterProvider = 0
	ClusterProvider_BareMetal   ClusterProvider = 1
	ClusterProvider_Aws         ClusterProvider = 2
	ClusterProvider_AliCloud    ClusterProvider = 3
)

// ClusterProvider to string
func (cp ClusterProvider) String() string {
	switch cp {
	case ClusterProvider_BareMetal:
		return "baremetal"
	case ClusterProvider_Aws:
		return "aws"
	case ClusterProvider_AliCloud:
		return "ali_cloud"
	default:
		return ""
	}
}

type ClusterStatus int32

const (
	ClusterStatus_UNSPECIFIED ClusterStatus = 0
	ClusterStatus_STARTING    ClusterStatus = 1 // Cluster init
	ClusterStatus_RUNNING     ClusterStatus = 2 // Network ready & storage ready & monitoring ready & toolkit ready
	ClusterStatus_STOPPING    ClusterStatus = 3
	ClusterStatus_STOPPED     ClusterStatus = 4
	ClusterStatus_DELETED     ClusterStatus = 5
)

// ClusterStatus to string
func (cs ClusterStatus) String() string {
	switch cs {
	case ClusterStatus_STARTING:
		return "starting"
	case ClusterStatus_RUNNING:
		return "running"
	case ClusterStatus_STOPPING:
		return "stopping"
	case ClusterStatus_STOPPED:
		return "stopped"
	case ClusterStatus_DELETED:
		return "deleted"
	default:
		return ""
	}
}

type ClusterLevel int32

const (
	ClusterLevel_UNSPECIFIED ClusterLevel = 0
	ClusterLevel_BASIC       ClusterLevel = 1
	ClusterLevel_STANDARD    ClusterLevel = 2
	ClusterLevel_ADVANCED    ClusterLevel = 3
)

// ClusterLevel to string
func (cl ClusterLevel) String() string {
	switch cl {
	case ClusterLevel_BASIC:
		return "basic"
	case ClusterLevel_STANDARD:
		return "standard"
	case ClusterLevel_ADVANCED:
		return "advanced"
	default:
		return ""
	}
}

type NodeRole int32

const (
	NodeRole_UNSPECIFIED NodeRole = 0
	NodeRole_MASTER      NodeRole = 1
	NodeRole_WORKER      NodeRole = 2
	NodeRole_EDGE        NodeRole = 3
)

// NodeRole to string
func (nr NodeRole) String() string {
	switch nr {
	case NodeRole_MASTER:
		return "master"
	case NodeRole_WORKER:
		return "worker"
	case NodeRole_EDGE:
		return "edge"
	default:
		return ""
	}
}

type NodeStatus int32

const (
	NodeStatus_UNSPECIFIED   NodeStatus = 0
	NodeStatus_NODE_READY    NodeStatus = 1
	NodeStatus_NODE_FINDING  NodeStatus = 2
	NodeStatus_NODE_CREATING NodeStatus = 3
	NodeStatus_NODE_PENDING  NodeStatus = 4
	NodeStatus_NODE_RUNNING  NodeStatus = 5
	NodeStatus_NODE_DELETING NodeStatus = 6
	NodeStatus_NODE_DELETED  NodeStatus = 7
	NodeStatus_NODE_ERROR    NodeStatus = 8
)

// NodeStatus to string
func (ns NodeStatus) String() string {
	switch ns {
	case NodeStatus_NODE_READY:
		return "node_ready"
	case NodeStatus_NODE_FINDING:
		return "node_finding"
	case NodeStatus_NODE_CREATING:
		return "node_creating"
	case NodeStatus_NODE_PENDING:
		return "node_pending"
	case NodeStatus_NODE_RUNNING:
		return "node_running"
	case NodeStatus_NODE_DELETING:
		return "node_deleting"
	case NodeStatus_NODE_DELETED:
		return "node_deleted"
	case NodeStatus_NODE_ERROR:
		return "node_error"
	default:
		return ""
	}
}

type NodeGroupType int32

const (
	NodeGroupType_UNSPECIFIED      NodeGroupType = 0
	NodeGroupType_NORMAL           NodeGroupType = 1
	NodeGroupType_HIGH_COMPUTATION NodeGroupType = 2
	NodeGroupType_GPU_ACCELERATERD NodeGroupType = 3
	NodeGroupType_HIGH_MEMORY      NodeGroupType = 4
	NodeGroupType_LARGE_HARD_DISK  NodeGroupType = 5
	NodeGroupType_LOAD_DISK        NodeGroupType = 6
)

// NodeGroupType to string
func (ngt NodeGroupType) String() string {
	switch ngt {
	case NodeGroupType_NORMAL:
		return "normal"
	case NodeGroupType_HIGH_COMPUTATION:
		return "high_computation"
	case NodeGroupType_GPU_ACCELERATERD:
		return "gpu_accelerated"
	case NodeGroupType_HIGH_MEMORY:
		return "high_memory"
	case NodeGroupType_LARGE_HARD_DISK:
		return "large_hard_disk"
	case NodeGroupType_LOAD_DISK:
		return "load_disk"
	default:
		return ""
	}
}

type NodeArchType int32

const (
	NodeArchType_UNSPECIFIED NodeArchType = 0
	NodeArchType_AMD64       NodeArchType = 1
	NodeArchType_ARM64       NodeArchType = 2
)

func (n NodeArchType) String() string {
	switch n {
	case NodeArchType_AMD64:
		return "amd64"
	case NodeArchType_ARM64:
		return "arm64"
	default:
		return ""
	}
}

func NodeArchTypeFromString(s string) NodeArchType {
	switch s {
	case "amd64":
		return NodeArchType_AMD64
	case "arm64":
		return NodeArchType_ARM64
	default:
		return 0
	}
}

type NodeGPUSpec int32

const (
	NodeGPUSpec_UNSPECIFIED NodeGPUSpec = 0
	NodeGPUSpec_NVIDIA_A10  NodeGPUSpec = 1
	NodeGPUSpec_NVIDIA_V100 NodeGPUSpec = 2
	NodeGPUSpec_NVIDIA_T4   NodeGPUSpec = 3
	NodeGPUSpec_NVIDIA_P100 NodeGPUSpec = 4
	NodeGPUSpec_NVIDIA_P4   NodeGPUSpec = 5
)

func (n NodeGPUSpec) String() string {
	switch n {
	case NodeGPUSpec_NVIDIA_A10:
		return "nvidia-a10"
	case NodeGPUSpec_NVIDIA_V100:
		return "nvidia-v100"
	case NodeGPUSpec_NVIDIA_T4:
		return "nvidia-t4"
	case NodeGPUSpec_NVIDIA_P100:
		return "nvidia-p100"
	case NodeGPUSpec_NVIDIA_P4:
		return "nvidia-p4"
	default:
		return ""
	}
}

// string to NodeGPUSpec
func NodeGPUSpecFromString(s string) NodeGPUSpec {
	switch s {
	case "nvidia-a10":
		return NodeGPUSpec_NVIDIA_A10
	case "nvidia-v100":
		return NodeGPUSpec_NVIDIA_V100
	case "nvidia-t4":
		return NodeGPUSpec_NVIDIA_T4
	case "nvidia-p100":
		return NodeGPUSpec_NVIDIA_P100
	case "nvidia-p4":
		return NodeGPUSpec_NVIDIA_P4
	default:
		return 0
	}
}

type ResourceType int32

const (
	ResourceType_UNSPECIFIED        ResourceType = 0
	ResourceType_VPC                ResourceType = 1
	ResourceType_SUBNET             ResourceType = 2
	ResourceType_INTERNET_GATEWAY   ResourceType = 3
	ResourceType_NAT_GATEWAY        ResourceType = 4
	ResourceType_ROUTE_TABLE        ResourceType = 5
	ResourceType_SECURITY_GROUP     ResourceType = 6
	ResourceType_LOAD_BALANCER      ResourceType = 7
	ResourceType_ELASTIC_IP         ResourceType = 8
	ResourceType_AVAILABILITY_ZONES ResourceType = 9
	ResourceType_KEY_PAIR           ResourceType = 10
	ResourceType_DATA_DEVICE        ResourceType = 11
	ResourceType_INSTANCE           ResourceType = 12
	ResourceType_REGION             ResourceType = 13
	ResourceType_GATEWAY_CLASS      ResourceType = 14
	ResourceType_STORAGE_CLASS      ResourceType = 15
)

// ResourceType to string
func (rt ResourceType) String() string {
	switch rt {
	case ResourceType_VPC:
		return "vpc"
	case ResourceType_SUBNET:
		return "subnet"
	case ResourceType_INTERNET_GATEWAY:
		return "internet_gateway"
	case ResourceType_NAT_GATEWAY:
		return "nat_gateway"
	case ResourceType_ROUTE_TABLE:
		return "route_table"
	case ResourceType_SECURITY_GROUP:
		return "security_group"
	case ResourceType_LOAD_BALANCER:
		return "load_balancer"
	case ResourceType_ELASTIC_IP:
		return "elastic_ip"
	case ResourceType_AVAILABILITY_ZONES:
		return "availability_zones"
	case ResourceType_KEY_PAIR:
		return "key_pair"
	case ResourceType_DATA_DEVICE:
		return "data_device"
	case ResourceType_INSTANCE:
		return "instance"
	case ResourceType_REGION:
		return "region"
	case ResourceType_GATEWAY_CLASS:
		return "gateway_class"
	case ResourceType_STORAGE_CLASS:
		return "storage_class"
	default:
		return ""
	}
}

type ResourceTypeKeyValue int32

const (
	ResourceTypeKeyValue_UNSPECIFIED      ResourceTypeKeyValue = 0
	ResourceTypeKeyValue_NAME             ResourceTypeKeyValue = 1
	ResourceTypeKeyValue_ACCESS           ResourceTypeKeyValue = 2
	ResourceTypeKeyValue_ZONE_ID          ResourceTypeKeyValue = 3
	ResourceTypeKeyValue_REGION_ID        ResourceTypeKeyValue = 4
	ResourceTypeKeyValue_ACCESS_PRIVATE   ResourceTypeKeyValue = 5
	ResourceTypeKeyValue_ACCESS_PUBLIC    ResourceTypeKeyValue = 6
	ResourceTypeKeyValue_EXTERNAL_GATEWAY ResourceTypeKeyValue = 7
	ResourceTypeKeyValue_INTERNAL_GATEWAY ResourceTypeKeyValue = 8
	ResourceTypeKeyValue_BLOCK_STORAGE    ResourceTypeKeyValue = 9
	ResourceTypeKeyValue_FILE_STORAGE     ResourceTypeKeyValue = 10
	ResourceTypeKeyValue_OBJECT_STORAGE   ResourceTypeKeyValue = 11
)

// ResourceTypeKeyValue to string
func (kv ResourceTypeKeyValue) String() string {
	switch kv {
	case ResourceTypeKeyValue_NAME:
		return "name"
	case ResourceTypeKeyValue_ACCESS:
		return "access"
	case ResourceTypeKeyValue_ZONE_ID:
		return "zone_id"
	case ResourceTypeKeyValue_REGION_ID:
		return "region_id"
	case ResourceTypeKeyValue_ACCESS_PRIVATE:
		return "access_private"
	case ResourceTypeKeyValue_ACCESS_PUBLIC:
		return "access_public"
	case ResourceTypeKeyValue_EXTERNAL_GATEWAY:
		return "external_gateway"
	case ResourceTypeKeyValue_INTERNAL_GATEWAY:
		return "internal_gateway"
	case ResourceTypeKeyValue_BLOCK_STORAGE:
		return "block_storage"
	case ResourceTypeKeyValue_FILE_STORAGE:
		return "file_storage"
	case ResourceTypeKeyValue_OBJECT_STORAGE:
		return "object_storage"
	default:
		return ""
	}
}

type IngressControllerRuleAccess int32

const (
	IngressControllerRuleAccess_UNSPECIFIED IngressControllerRuleAccess = 0
	IngressControllerRuleAccess_PRIVATE     IngressControllerRuleAccess = 1
	IngressControllerRuleAccess_PUBLIC      IngressControllerRuleAccess = 2
)

type NodeErrorType int32

const (
	NodeErrorType_UNSPECIFIED          NodeErrorType = 0
	NodeErrorType_INFRASTRUCTURE_ERROR NodeErrorType = 1
	NodeErrorType_CLUSTER_ERROR        NodeErrorType = 2
)

type Cluster struct {
	Id                     int64                    `gorm:"column:id;primaryKey;AUTO_INCREMENT" json:"id,omitempty"`
	Name                   string                   `gorm:"column:name;default:'';NOT NULL" json:"name,omitempty"`
	ApiServerAddress       string                   `gorm:"column:api_server_address;default:'';NOT NULL" json:"api_server_address,omitempty"`
	ApiServerPort          string                   `gorm:"column:api_server_port;default:'';NOT NULL" json:"api_server_port,omitempty"`
	ImageRepo              string                   `gorm:"column:image_repo;default:'';NOT NULL" json:"image_repo,omitempty"`
	Config                 string                   `gorm:"column:config;default:'';NOT NULL" json:"config,omitempty"`
	Status                 ClusterStatus            `gorm:"column:status;default:0;NOT NULL" json:"status,omitempty"`
	Provider               ClusterProvider          `gorm:"column:provider;default:0;NOT NULL" json:"provider,omitempty"`
	Level                  ClusterLevel             `gorm:"column:level;default:0;NOT NULL" json:"level,omitempty"`
	PublicKey              string                   `gorm:"column:public_key;default:'';NOT NULL" json:"public_key,omitempty"`
	PrivateKey             string                   `gorm:"column:private_key;default:'';NOT NULL" json:"private_key,omitempty"`
	Region                 string                   `gorm:"column:region;default:'';NOT NULL" json:"region,omitempty"`
	UserId                 int64                    `gorm:"column:user_id;default:0;NOT NULL" json:"user_id,omitempty"`
	AccessId               string                   `gorm:"column:access_id;default:'';NOT NULL" json:"access_id,omitempty"`
	AccessKey              string                   `gorm:"column:access_key;default:'';NOT NULL" json:"access_key,omitempty"`
	ResroucePath           string                   `gorm:"column:resrouce_path;default:'';NOT NULL" json:"resrouce_path,omitempty"`
	NodeStartIp            string                   `gorm:"column:node_start_ip;default:'';NOT NULL" json:"node_start_ip,omitempty"`
	NodeEndIp              string                   `gorm:"column:node_end_ip;default:'';NOT NULL" json:"node_end_ip,omitempty"`
	KuberentesVersion      string                   `gorm:"column:kuberentes_version;default:'';NOT NULL" json:"kuberentes_version,omitempty"`
	ContainerdVersion      string                   `gorm:"column:containerd_version;default:'';NOT NULL" json:"containerd_version,omitempty"`
	RuncVersion            string                   `gorm:"column:runc_version;default:'';NOT NULL" json:"runc_version,omitempty"`
	CiliumVersion          string                   `gorm:"column:cilium_version;default:'';NOT NULL" json:"cilium_version,omitempty"`
	ClusterInfo            string                   `gorm:"column:cluster_info;default:'';NOT NULL" json:"cluster_info,omitempty"`
	Domain                 string                   `gorm:"column:domain;default:'';NOT NULL" json:"domain,omitempty"`
	VpcCidr                string                   `gorm:"column:vpc_cidr;default:'';NOT NULL" json:"vpc_cidr,omitempty"`
	ServiceCidr            string                   `gorm:"column:service_cidr;default:'';NOT NULL" json:"service_cidr,omitempty"`
	PodCidr                string                   `gorm:"column:pod_cidr;default:'';NOT NULL" json:"pod_cidr,omitempty"`
	KubeConfigPath         string                   `gorm:"column:kube_config_path;default:'';NOT NULL" json:"kube_config_path,omitempty"`
	NodeGroups             []*NodeGroup             `gorm:"-" json:"node_groups,omitempty"`
	Nodes                  []*Node                  `gorm:"-" json:"nodes,omitempty"`
	CloudResources         []*CloudResource         `gorm:"-" json:"cloud_resources,omitempty"`
	IngressControllerRules []*IngressControllerRule `gorm:"-" json:"ingress_controller_rules,omitempty"`
}

type NodeGroup struct {
	Id           string        `gorm:"column:id;primaryKey;NOT NULL" json:"id,omitempty"`
	Name         string        `gorm:"column:name;default:'';NOT NULL" json:"name,omitempty"`
	Type         NodeGroupType `gorm:"column:type;default:0;NOT NULL" json:"type,omitempty"`
	Os           string        `gorm:"column:os;default:'';NOT NULL" json:"os,omitempty"`
	Platform     string        `gorm:"column:platform;default:'';NOT NULL" json:"platform,omitempty"`
	Arch         NodeArchType  `gorm:"column:arch;default:0;NOT NULL" json:"arch,omitempty"`
	Cpu          int32         `gorm:"column:cpu;default:0;NOT NULL" json:"cpu,omitempty"`
	Memory       int32         `gorm:"column:memory;default:0;NOT NULL" json:"memory,omitempty"`
	Gpu          int32         `gorm:"column:gpu;default:0;NOT NULL" json:"gpu,omitempty"`
	GpuSpec      NodeGPUSpec   `gorm:"column:gpu_spec;default:0;NOT NULL" json:"gpu_spec,omitempty"`
	MinSize      int32         `gorm:"column:min_size;default:0;NOT NULL" json:"min_size,omitempty"`
	MaxSize      int32         `gorm:"column:max_size;default:0;NOT NULL" json:"max_size,omitempty"`
	TargetSize   int32         `gorm:"column:target_size;default:0;NOT NULL" json:"target_size,omitempty"`
	NodePrice    float32       `gorm:"column:node_price;default:0;NOT NULL" json:"node_price,omitempty"`
	PodPrice     float32       `gorm:"column:pod_price;default:0;NOT NULL" json:"pod_price,omitempty"`
	SubnetIpCidr string        `gorm:"column:subnet_ip_cidr;default:'';NOT NULL" json:"subnet_ip_cidr,omitempty"`
	PodIpCidr    string        `gorm:"column:pod_ip_cidr;default:'';NOT NULL" json:"pod_ip_cidr,omitempty"`
	ClusterId    int64         `gorm:"column:cluster_id;default:0;NOT NULL" json:"cluster_id,omitempty"`
}

type Node struct {
	Id                int64         `gorm:"column:id;primaryKey;AUTO_INCREMENT" json:"id,omitempty"`
	Name              string        `gorm:"column:name;default:'';NOT NULL" json:"name,omitempty"`
	Labels            string        `gorm:"column:labels;default:'';NOT NULL" json:"labels,omitempty"`
	Ip                string        `gorm:"column:ip;default:'';NOT NULL" json:"ip,omitempty"`
	User              string        `gorm:"column:user;default:'';NOT NULL" json:"user,omitempty"`
	Role              NodeRole      `gorm:"column:role;default:0;NOT NULL" json:"role,omitempty"`
	Status            NodeStatus    `gorm:"column:status;default:0;NOT NULL" json:"status,omitempty"`
	InstanceId        string        `gorm:"column:instance_id;default:'';NOT NULL" json:"instance_id,omitempty"`
	ImageId           string        `gorm:"column:image_id;default:'';NOT NULL" json:"image_id,omitempty"`
	BackupInstanceIds string        `gorm:"column:backup_instance_ids;default:'';NOT NULL" json:"backup_instance_ids,omitempty"`
	InstanceType      string        `gorm:"column:instance_type;default:'';NOT NULL" json:"instance_type,omitempty"`
	SystemDiskSize    int32         `gorm:"column:system_disk_size;default:0;NOT NULL" json:"system_disk_size,omitempty"`
	SystemDiskName    string        `gorm:"column:system_disk_name;default:'';NOT NULL" json:"system_disk_name,omitempty"`
	DataDiskSize      int32         `gorm:"column:data_disk_size;default:0;NOT NULL" json:"data_disk_size,omitempty"`
	DataDiskName      string        `gorm:"column:data_disk_name;default:'';NOT NULL" json:"data_disk_name,omitempty"`
	ClusterId         int64         `gorm:"column:cluster_id;default:0;NOT NULL" json:"cluster_id,omitempty"`
	NodeGroupId       string        `gorm:"column:node_group_id;default:'';NOT NULL" json:"node_group_id,omitempty"`
	NodeInfo          string        `gorm:"column:node_info;default:'';NOT NULL" json:"node_info,omitempty"`
	ErrorType         NodeErrorType `gorm:"column:error_type;default:0;NOT NULL" json:"error_type,omitempty"`
	ErrorMessage      string        `gorm:"column:error_message;default:'';NOT NULL" json:"error_message,omitempty"`
}

type CloudResource struct {
	Id           string       `gorm:"column:id;primaryKey;NOT NULL" json:"id,omitempty"`
	Name         string       `gorm:"column:name;default:'';NOT NULL" json:"name,omitempty"`
	RefId        string       `gorm:"column:ref_id;default:'';NOT NULL" json:"ref_id,omitempty"`
	AssociatedId string       `gorm:"column:associated_id;default:'';NOT NULL" json:"associated_id,omitempty"`
	Type         ResourceType `gorm:"column:type;default:0;NOT NULL" json:"type,omitempty"`
	Tags         string       `gorm:"column:tags;default:'';NOT NULL" json:"tags,omitempty"`
	Value        string       `gorm:"column:value;default:'';NOT NULL" json:"value,omitempty"`
	ClusterId    int64        `gorm:"column:cluster_id;default:0;NOT NULL" json:"cluster_id,omitempty"`
}

type IngressControllerRule struct {
	Id        string                      `gorm:"column:id;primaryKey;NOT NULL" json:"id,omitempty"`
	Name      string                      `gorm:"column:name;default:'';NOT NULL" json:"name,omitempty"`
	StartPort int32                       `gorm:"column:start_port;default:0;NOT NULL" json:"start_port,omitempty"`
	EndPort   int32                       `gorm:"column:end_port;default:0;NOT NULL" json:"end_port,omitempty"`
	Protocol  string                      `gorm:"column:protocol;default:'';NOT NULL" json:"protocol,omitempty"`
	IpCidr    string                      `gorm:"column:ip_cidr;default:'';NOT NULL" json:"ip_cidr,omitempty"`
	Access    IngressControllerRuleAccess `gorm:"column:access;default:0;NOT NULL" json:"access,omitempty"`
	ClusterId int64                       `gorm:"column:cluster_id;default:0;NOT NULL" json:"cluster_id,omitempty"`
}

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
	ReloadCluster(context.Context, *Cluster) error
}

func WithCluster(ctx context.Context, cluster *Cluster) context.Context {
	return context.WithValue(ctx, ClusterKey, cluster)
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

func (c *Cluster) IsDeleted() bool {
	return false
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
		resource.Id = uuid.NewString()
	}
	c.CloudResources = append(c.CloudResources, resource)
}

func (c *Cluster) GetCloudResourceByName(resourceType ResourceType, name string) *CloudResource {
	for _, resource := range c.CloudResources {
		if resource.Type == resourceType && resource.Name == name {
			return resource
		}
	}
	return nil
}

func (c *Cluster) GetCloudResourceByID(resourceType ResourceType, id string) *CloudResource {
	resource := c.getCloudResourceByID(c.GetCloudResource(resourceType), id)
	if resource != nil {
		return resource
	}
	return nil
}

func (c *Cluster) GetCloudResourceByRefID(resourceType ResourceType, refID string) *CloudResource {
	for _, resource := range c.CloudResources {
		if resource.Type == resourceType && resource.RefId == refID {
			return resource
		}
	}
	return nil
}

func (c *Cluster) getCloudResourceByID(cloudResources []*CloudResource, id string) *CloudResource {
	for _, resource := range cloudResources {
		if resource.Id == id {
			return resource
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
func (c *Cluster) GetCloudResourceByTags(resourceType ResourceType, tagKeyValues map[ResourceTypeKeyValue]any) []*CloudResource {
	cloudResources := make([]*CloudResource, 0)
	for _, resource := range c.GetCloudResource(resourceType) {
		if resource.Tags == "" {
			continue
		}
		resourceTagsMap := c.DecodeTags(resource.Tags)
		match := true
		for key, value := range tagKeyValues {
			val, ok := resourceTagsMap[key]
			if !ok {
				match = false
				break
			}
			if resourceTypeKeyValue, ok := value.(ResourceTypeKeyValue); ok {
				if int32(resourceTypeKeyValue) != cast.ToInt32(val) {
					match = false
					break
				}
				continue
			}
			if cast.ToString(val) != cast.ToString(value) {
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

func (c *Cluster) GetCloudResourceByTagsSingle(resourceType ResourceType, tagKeyValues map[ResourceTypeKeyValue]any) *CloudResource {
	resources := c.GetCloudResourceByTags(resourceType, tagKeyValues)
	if len(resources) == 0 {
		return nil
	}
	return resources[0]
}

func (c *Cluster) EncodeTags(tags map[ResourceTypeKeyValue]any) string {
	if tags == nil {
		return ""
	}
	jsonBytes, _ := json.Marshal(tags)
	return string(jsonBytes)
}

func (c *Cluster) DecodeTags(tags string) map[ResourceTypeKeyValue]any {
	tagsMap := make(map[ResourceTypeKeyValue]any)
	if tags == "" {
		return tagsMap
	}
	json.Unmarshal([]byte(tags), &tagsMap)
	return tagsMap
}

// delete cloud resource by resourceType
func (c *Cluster) DeleteCloudResource(resourceType ResourceType) {
	cloudResources := make([]*CloudResource, 0)
	for _, resources := range c.CloudResources {
		if resources.Type != resourceType {
			cloudResources = append(cloudResources, resources)
		}
	}
	c.CloudResources = cloudResources
}

// delete cloud resource by resourceType and id
func (c *Cluster) DeleteCloudResourceByID(resourceType ResourceType, id string) {
	cloudResources := make([]*CloudResource, 0)
	for _, resources := range c.CloudResources {
		if resources.Type == resourceType && resources.Id == id {
			continue
		}
		cloudResources = append(cloudResources, resources)
	}
	c.CloudResources = cloudResources
}

// delete cloud resource by resourceType and refID
func (c *Cluster) DeleteCloudResourceByRefID(resourceType ResourceType, refID string) {
	cloudResources := make([]*CloudResource, 0)
	for _, resources := range c.CloudResources {
		if resources.Type == resourceType && resources.RefId == refID {
			continue
		}
		cloudResources = append(cloudResources, resources)
	}
	c.CloudResources = cloudResources
}

// delete cloud resource by resourceType and tag value and tag key
func (c *Cluster) DeleteCloudResourceByTags(resourceType ResourceType, tagKeyValues ...ResourceTypeKeyValue) {
	cloudResources := make([]*CloudResource, 0)
	for _, resource := range c.CloudResources {
		if resource.Tags == "" {
			cloudResources = append(cloudResources, resource)
			continue
		}
		if resource.Type != resourceType {
			cloudResources = append(cloudResources, resource)
			continue
		}
		match := true
		resourceTagsMap := c.DecodeTags(resource.Tags)
		for i := 0; i < len(tagKeyValues); i += 2 {
			tagKey := tagKeyValues[i]
			tagValue := tagKeyValues[i+1]
			if resourceTagsMap[tagKey] != tagValue {
				match = false
				break
			}
		}
		if match {
			continue
		}
		cloudResources = append(cloudResources, resource)
	}
	c.CloudResources = cloudResources
}

func (c *Cluster) EncodeNodeGroup(nodeGroup *NodeGroup) string {
	return strings.Join([]string{
		strings.ToUpper(nodeGroup.Os),
		strings.ToUpper(nodeGroup.Platform),
		nodeGroup.Arch.String(),
		fmt.Sprintf("%d-%d-%d", nodeGroup.Cpu, nodeGroup.Memory, nodeGroup.Gpu),
		nodeGroup.GpuSpec.String(),
	}, "-")
}

func (c *Cluster) DecodeNodeGroup(nodeGroup string) *NodeGroup {
	nodeGroupSlice := strings.Split(nodeGroup, "-")
	if len(nodeGroupSlice) != 5 {
		return nil
	}
	return &NodeGroup{
		Os:       strings.ToLower(nodeGroupSlice[0]),
		Platform: strings.ToLower(nodeGroupSlice[1]),
		Arch:     NodeArchTypeFromString(nodeGroupSlice[2]),
		Cpu:      cast.ToInt32(nodeGroupSlice[3]),
		Memory:   cast.ToInt32(nodeGroupSlice[4]),
		Gpu:      cast.ToInt32(nodeGroupSlice[5]),
		GpuSpec:  NodeGPUSpecFromString(nodeGroupSlice[6]),
	}
}

func (c ClusterProvider) IsCloud() bool {
	return c != ClusterProvider_BareMetal
}

func (c *Cluster) GetNodeGroup(nodeGroupId string) *NodeGroup {
	for _, nodeGroup := range c.NodeGroups {
		if nodeGroup.Id == nodeGroupId {
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

func (c *Cluster) DistributeNodePrivateSubnets(nodeIndex int) *CloudResource {
	tags := c.GetTags()
	tags[ResourceTypeKeyValue_ACCESS] = ResourceTypeKeyValue_ACCESS_PRIVATE
	subnets := c.GetCloudResourceByTags(ResourceType_SUBNET, tags)
	if len(subnets) == 0 {
		return nil
	}
	nodeSize := len(c.Nodes)
	subnetsSize := len(subnets)
	if nodeSize <= subnetsSize {
		return subnets[nodeIndex%subnetsSize]
	}
	interval := nodeSize / subnetsSize
	return subnets[(nodeIndex/interval)%subnetsSize]
}

func (c *Cluster) GetTags() map[ResourceTypeKeyValue]any {
	return make(map[ResourceTypeKeyValue]any)
}

func (c *Cluster) CreateCluster() bool {
	return c.Status == ClusterStatus_STARTING
}

func (c *Cluster) UpdateCluster() bool {
	return c.Status == ClusterStatus_RUNNING
}

func (c *Cluster) DeleteCluster() bool {
	return c.Status == ClusterStatus_STOPPING || c.Status == ClusterStatus_DELETED
}

func (g *NodeGroup) CreateOrUpdateNodeGroup() bool {
	return g.TargetSize > 0
}

func (g *NodeGroup) DeleteNodeGroup() bool {
	return g.TargetSize == 0
}

func (n *Node) CreateNode() bool {
	return n.Status == NodeStatus_NODE_CREATING
}

func (n *Node) UpdateNode() bool {
	return n.Status == NodeStatus_NODE_RUNNING || n.Status == NodeStatus_NODE_PENDING
}

func (n *Node) DeleteNode() bool {
	return n.Status == NodeStatus_NODE_DELETING || n.Status == NodeStatus_NODE_DELETED
}

func (c *Cluster) GetVpcName() string {
	return fmt.Sprintf("%s-vpc", c.Name)
}

func (c *Cluster) GetkeyPairName() string {
	return fmt.Sprintf("%s-keypair", c.Name)
}

func (c *Cluster) GetSubnetName(zoneId string) string {
	return fmt.Sprintf("%s-%s-subnet", c.Name, zoneId)
}

func (c *Cluster) GetEipName(zoneId string) string {
	return fmt.Sprintf("%s-%s-eip", c.Name, zoneId)
}

func (c *Cluster) GetNatgatewayName(zoneId string) string {
	return fmt.Sprintf("%s-%s-natgateway", c.Name, zoneId)
}

func (c *Cluster) GetSecurityGroupName() string {
	return fmt.Sprintf("%s-securitygroup", c.Name)
}

func (c *Cluster) GetRouteTableName(zoneId string) string {
	return fmt.Sprintf("%s-%s-route-table", c.Name, zoneId)
}

func (c *Cluster) GetPublicRouteTableName() string {
	return fmt.Sprintf("%s-public-route-table", c.Name)
}

func (c *Cluster) GetLoadBalancerName() string {
	return strings.ReplaceAll(fmt.Sprintf("%s-slb", c.Name), "_", "-")
}

func GetCluster(ctx context.Context) *Cluster {
	cluster, ok := ctx.Value(ClusterKey).(*Cluster)
	if !ok {
		return nil
	}
	return cluster
}

func (c *Cluster) GetLabels() map[string]string {
	return map[string]string{
		"cluster": c.Name,
	}
}

func (c *Cluster) SettingClusterLevel(clusterLevel *confPkg.Level) {
	var maxNodeNumber int32 = 0
	for _, nodeGroup := range c.NodeGroups {
		maxNodeNumber += nodeGroup.TargetSize
	}
	var setClusterLevel ClusterLevel = ClusterLevel_UNSPECIFIED
	if maxNodeNumber < clusterLevel.Basic {
		setClusterLevel = ClusterLevel_BASIC
	}
	if maxNodeNumber < clusterLevel.Advanced && maxNodeNumber >= clusterLevel.Basic {
		setClusterLevel = ClusterLevel_ADVANCED
	}
	if maxNodeNumber >= clusterLevel.Advanced {
		setClusterLevel = ClusterLevel_STANDARD
	}
	if c.Level != setClusterLevel && setClusterLevel != ClusterLevel_UNSPECIFIED {
		c.Level = setClusterLevel
	}
}

func (c *Cluster) SettingClusterAvailabilityZoneByClusterLevel(zones []*CloudResource) {
	zoneNumber := len(zones)
	if c.Level == ClusterLevel_BASIC {
		zoneNumber = 1
	}
	if c.Level == ClusterLevel_STANDARD {
		zoneNumber = int(math.Ceil(float64(zoneNumber) / 2))
	}
	if zoneNumber <= len(c.GetCloudResource(ResourceType_AVAILABILITY_ZONES)) {
		return
	}
	needNewZoneNumber := zoneNumber - len(c.GetCloudResource(ResourceType_AVAILABILITY_ZONES))
	for _, zone := range zones {
		ok := false
		for _, v := range c.GetCloudResource(ResourceType_AVAILABILITY_ZONES) {
			if v.RefId == zone.RefId {
				ok = true
			}
		}
		if !ok && needNewZoneNumber > 0 {
			c.AddCloudResource(zone)
			needNewZoneNumber--
		}
	}
}

func (c *Cluster) SettingDefaultNodeGroup(nodegroupConfig *confPkg.NodeGroupConfig) {
	if c.NodeGroups == nil {
		c.NodeGroups = make([]*NodeGroup, 0)
	}
	if c.Nodes == nil {
		c.Nodes = make([]*Node, 0)
	}
	if !c.Provider.IsCloud() {
		return
	}
	c.NodeGroups = append(c.NodeGroups, &NodeGroup{
		Id:         uuid.NewString(),
		Name:       c.Name + "-default",
		ClusterId:  c.Id,
		Type:       NodeGroupType_NORMAL,
		TargetSize: nodegroupConfig.TargetSize,
		MaxSize:    nodegroupConfig.MaxSize,
		MinSize:    nodegroupConfig.MinSize,
		Arch:       NodeArchType_AMD64,
		Cpu:        nodegroupConfig.Cpu,
		Memory:     nodegroupConfig.Memory,
	})
	c.Nodes = append(c.Nodes, &Node{
		Name:           "node1",
		Status:         NodeStatus_NODE_FINDING,
		Role:           NodeRole_MASTER,
		NodeGroupId:    c.NodeGroups[0].Id,
		SystemDiskSize: nodegroupConfig.DiskSize,
		ClusterId:      c.Id,
	}, &Node{
		Name:           "node2",
		Status:         NodeStatus_NODE_FINDING,
		Role:           NodeRole_WORKER,
		NodeGroupId:    c.NodeGroups[0].Id,
		SystemDiskSize: nodegroupConfig.DiskSize,
		ClusterId:      c.Id,
	}, &Node{
		Name:           "node3",
		Status:         NodeStatus_NODE_FINDING,
		Role:           NodeRole_WORKER,
		NodeGroupId:    c.NodeGroups[0].Id,
		SystemDiskSize: nodegroupConfig.DiskSize,
		ClusterId:      c.Id,
	})
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
		if rule.Access {
			clusterIngressControllerRule.Access = IngressControllerRuleAccess_PUBLIC
		} else {
			clusterIngressControllerRule.Access = IngressControllerRuleAccess_PRIVATE
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

func (c *Cluster) SetStatus(status ClusterStatus) {
	c.Status = status
}

func (c *Cluster) SetRegion(region string) {
	c.Region = region
}

func (c *Cluster) SetNodeStatus(fromStatus, toStatus NodeStatus) {
	for _, node := range c.Nodes {
		if node.Status == fromStatus {
			node.SetNodeStatus(toStatus)
		}
	}
}

func (c *Cluster) GetCpuCount() int32 {
	var cpuCount int32 = 0
	for _, nodeGroup := range c.NodeGroups {
		cpuCount += nodeGroup.Cpu
	}
	return cpuCount
}

func (c *Cluster) GetGpuCount() int32 {
	var gpuCount int32 = 0
	for _, nodeGroup := range c.NodeGroups {
		gpuCount += nodeGroup.Gpu
	}
	return gpuCount
}

func (c *Cluster) GetMemoryCount() int32 {
	var memoryCount int32 = 0
	for _, nodeGroup := range c.NodeGroups {
		memoryCount += nodeGroup.Memory
	}
	return memoryCount
}

func (c *Cluster) GetDiskSizeCount() int32 {
	var diskSizeCount int32 = 0
	for _, node := range c.Nodes {
		diskSizeCount += node.DataDiskSize + node.SystemDiskSize
	}
	return diskSizeCount
}

func (ng *NodeGroup) SetTargetSize(size int32) {
	ng.TargetSize = size
}

func (n *Node) SetNodeStatus(status NodeStatus) {
	n.Status = status
}

func (c *Cluster) generateNodeLables(nodeGroup *NodeGroup) string {
	lableMap := make(map[string]string)
	lableMap["cluster"] = c.Name
	lableMap["cluster_id"] = cast.ToString(c.Id)
	lableMap["cluster_type"] = c.Provider.String()
	lableMap["region"] = c.Region
	lableMap["nodegroup"] = nodeGroup.Name
	lableMap["nodegroup_type"] = nodeGroup.Type.String()
	lablebytes, _ := json.Marshal(lableMap)
	return string(lablebytes)
}

func (uc *ClusterUsecase) NodeGroupIncreaseSize(ctx context.Context, cluster *Cluster, nodeGroup *NodeGroup, size int32) error {
	for range make([]struct{}, size) {
		node := &Node{
			Name:        fmt.Sprintf("%s-%s", cluster.Name, uuid.New().String()),
			Role:        NodeRole_WORKER,
			Status:      NodeStatus_NODE_CREATING,
			ClusterId:   cluster.Id,
			NodeGroupId: nodeGroup.Id,
		}
		cluster.Nodes = append(cluster.Nodes, node)
	}
	return nil
}

func (uc *ClusterUsecase) DeleteNodes(ctx context.Context, cluster *Cluster, nodes []*Node) error {
	for _, node := range nodes {
		for i, n := range cluster.Nodes {
			if n.Id == node.Id {
				cluster.Nodes = append(cluster.Nodes[:i], cluster.Nodes[i+1:]...)
				break
			}
		}
	}
	return nil
}

func (uc *ClusterUsecase) NodeGroupTemplateNodeInfo(ctx context.Context, cluster *Cluster, nodeGroup *NodeGroup) (*Node, error) {
	return &Node{
		Name:        fmt.Sprintf("%s-%s", cluster.Name, uuid.New().String()),
		Role:        NodeRole_WORKER,
		Status:      NodeStatus_NODE_CREATING,
		ClusterId:   cluster.Id,
		NodeGroupId: nodeGroup.Id,
		Labels:      cluster.generateNodeLables(nodeGroup),
	}, nil
}

func (uc *ClusterUsecase) Cleanup(ctx context.Context) error {
	return nil
}

func (uc *ClusterUsecase) Refresh(ctx context.Context) error {
	cluster, err := uc.GetCurrentCluster(ctx)
	if err != nil {
		return err
	}
	err = uc.clusterRuntime.ReloadCluster(ctx, cluster)
	if err != nil {
		return err
	}
	return nil
}

func (uc *ClusterUsecase) GetCurrentCluster(ctx context.Context) (*Cluster, error) {
	cluster, err := uc.clusterData.GetByName(ctx, uc.conf.Cluster.Name)
	if err != nil {
		return nil, err
	}
	err = uc.clusterRuntime.CurrentCluster(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (uc *ClusterUsecase) GetClusterStatus() []ClusterStatus {
	return []ClusterStatus{
		ClusterStatus_STARTING,
		ClusterStatus_RUNNING,
		ClusterStatus_STOPPING,
		ClusterStatus_STOPPED,
		ClusterStatus_DELETED,
	}
}

func (uc *ClusterUsecase) GetClusterProviders() []ClusterProvider {
	return []ClusterProvider{
		ClusterProvider_BareMetal,
		ClusterProvider_Aws,
		ClusterProvider_AliCloud,
	}
}

func (uc *ClusterUsecase) GetClusterLevels() []ClusterLevel {
	return []ClusterLevel{
		ClusterLevel_BASIC,
		ClusterLevel_ADVANCED,
		ClusterLevel_STANDARD,
	}
}

func (uc *ClusterUsecase) GetNodeRoles() []NodeRole {
	return []NodeRole{
		NodeRole_MASTER,
		NodeRole_WORKER,
		NodeRole_EDGE,
	}
}

func (uc *ClusterUsecase) GetNodeStatuses() []NodeStatus {
	return []NodeStatus{
		NodeStatus_NODE_READY,
		NodeStatus_NODE_FINDING,
		NodeStatus_NODE_CREATING,
		NodeStatus_NODE_PENDING,
		NodeStatus_NODE_RUNNING,
		NodeStatus_NODE_DELETING,
		NodeStatus_NODE_DELETED,
		NodeStatus_NODE_ERROR,
	}
}

func (uc *ClusterUsecase) GetNodeGroupTypes() []NodeGroupType {
	return []NodeGroupType{
		NodeGroupType_NORMAL,
		NodeGroupType_HIGH_COMPUTATION,
		NodeGroupType_GPU_ACCELERATERD,
		NodeGroupType_HIGH_MEMORY,
		NodeGroupType_LARGE_HARD_DISK,
		NodeGroupType_LOAD_DISK,
	}
}

func (uc *ClusterUsecase) GetResourceTypes() []ResourceType {
	return []ResourceType{
		ResourceType_VPC,
		ResourceType_SUBNET,
		ResourceType_INTERNET_GATEWAY,
		ResourceType_NAT_GATEWAY,
		ResourceType_ROUTE_TABLE,
		ResourceType_SECURITY_GROUP,
		ResourceType_LOAD_BALANCER,
		ResourceType_ELASTIC_IP,
		ResourceType_AVAILABILITY_ZONES,
		ResourceType_KEY_PAIR,
		ResourceType_DATA_DEVICE,
		ResourceType_INSTANCE,
		ResourceType_REGION,
		ResourceType_GATEWAY_CLASS,
		ResourceType_STORAGE_CLASS,
	}
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
	if cluster.ImageRepo == "" {
		cluster.ImageRepo = uc.conf.Cluster.ImageRepository
	}
	return uc.clusterData.Save(ctx, cluster)
}

func (uc *ClusterUsecase) GetRegions(ctx context.Context, cluster *Cluster) ([]*CloudResource, error) {
	if cluster.Provider == ClusterProvider_BareMetal {
		return []*CloudResource{}, nil
	}
	err := uc.clusterInfrastructure.GetRegions(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return cluster.GetCloudResource(ResourceType_REGION), nil
}

func (uc *ClusterUsecase) StartCluster(ctx context.Context, clusterId int64) error {
	cluster, err := uc.Get(ctx, clusterId)
	if err != nil {
		return err
	}
	if cluster == nil || cluster.Id == 0 {
		return ErrClusterNotFound
	}
	if cluster.Status != ClusterStatus_UNSPECIFIED && cluster.Status != ClusterStatus_STOPPED {
		return errors.New("cluster is not in stopped state")
	}
	cluster.SetStatus(ClusterStatus_STARTING)
	err = uc.Apply(cluster)
	if err != nil {
		return err
	}
	return nil
}

func (uc *ClusterUsecase) StopCluster(ctx context.Context, clusterId int64) error {
	cluster, err := uc.Get(ctx, clusterId)
	if err != nil {
		return err
	}
	if cluster == nil || cluster.Id == 0 {
		return errors.New("cluster not found")
	}
	if cluster.Status != ClusterStatus_UNSPECIFIED && cluster.Status != ClusterStatus_RUNNING {
		return errors.New("cluster is not in running state")
	}
	cluster.SetStatus(ClusterStatus_STOPPING)
	err = uc.Apply(cluster)
	if err != nil {
		return err
	}
	return nil
}

// Start the cluster handler server
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
				return err
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
		lock.Unlock()
		if err != nil {
			uc.log.Errorf("cluster handle event error: %v", err)
		}
		err = uc.clusterData.Save(ctx, cluster)
		if err != nil {
			uc.log.Errorf("cluster save error: %v", err)
		}
		cluster.SetStatus(ClusterStatus_RUNNING)
	}()
	if cluster.IsDeleted() {
		for _, node := range cluster.Nodes {
			if node.Status == NodeStatus_UNSPECIFIED || node.Status == NodeStatus_NODE_DELETED {
				continue
			}
			node.SetNodeStatus(NodeStatus_NODE_DELETING)
		}
		err = uc.clusterRuntime.ReloadCluster(ctx, cluster)
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
	cluster.SettingClusterLevel(uc.conf.Cluster.Level)
	if cluster.Provider.IsCloud() {
		getErr := uc.clusterInfrastructure.GetZones(ctx, cluster)
		if getErr != nil {
			return getErr
		}
		cluster.SettingClusterAvailabilityZoneByClusterLevel(cluster.GetCloudResource(ResourceType_AVAILABILITY_ZONES))
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
	err = uc.clusterRuntime.ReloadCluster(ctx, cluster)
	if err != nil {
		return err
	}
	if cluster.Provider.IsCloud() {
		err = uc.clusterInfrastructure.ManageNodeResource(ctx, cluster)
		if err != nil {
			return err
		}
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
	if len(cluster.NodeGroups) == 0 {
		cluster.SettingDefaultNodeGroup(uc.conf.Cluster.NodegroupConfig)
		cluster.settingDefatultIngressRules(uc.conf.Cluster.IngressRules)
	}
	cluster.SettingClusterLevel(uc.conf.Cluster.Level)
	if cluster.Provider.IsCloud() {
		err := uc.clusterInfrastructure.GetZones(ctx, cluster)
		if err != nil {
			return err
		}
		cluster.SettingClusterAvailabilityZoneByClusterLevel(cluster.GetCloudResource(ResourceType_AVAILABILITY_ZONES))
		err = uc.clusterInfrastructure.CreateCloudBasicResource(ctx, cluster)
		if err != nil {
			return err
		}
	}
	err := uc.clusterInfrastructure.GetNodesSystemInfo(ctx, cluster)
	if err != nil {
		return err
	}
	if !cluster.Provider.IsCloud() {
		cluster.SetNodeStatus(NodeStatus_UNSPECIFIED, NodeStatus_NODE_READY)
		nodeGroupId := ""
		for _, nodeGroup := range cluster.NodeGroups {
			if nodeGroup.Cpu >= uc.conf.Cluster.NodegroupConfig.GetCpu() && nodeGroup.Memory >= uc.conf.Cluster.NodegroupConfig.GetMemory() {
				nodeGroupId = nodeGroup.Id
				break
			}
		}
		if nodeGroupId == "" {
			return errors.New("no node group found")
		}
		for _, node := range cluster.Nodes {
			if node.NodeGroupId == nodeGroupId {
				node.Status = NodeStatus_NODE_FINDING
				break
			}
		}
	}
	cluster.SetNodeStatus(NodeStatus_NODE_FINDING, NodeStatus_NODE_CREATING)
	if cluster.Provider.IsCloud() {
		err = uc.clusterInfrastructure.ManageNodeResource(ctx, cluster)
		if err != nil {
			return err
		}
	}
	cluster.SetNodeStatus(NodeStatus_NODE_CREATING, NodeStatus_NODE_PENDING)
	err = uc.clusterInfrastructure.Install(ctx, cluster)
	if err != nil {
		return err
	}
	cluster.SetNodeStatus(NodeStatus_NODE_PENDING, NodeStatus_NODE_RUNNING)
	return nil
}

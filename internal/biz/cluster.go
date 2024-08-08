package biz

import (
	"context"
	"fmt"

	"github.com/f-rambo/ocean/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type Cluster struct {
	ID               int64         `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name             string        `json:"name" gorm:"column:name; default:''; NOT NULL"` // *
	ServerVersion    string        `json:"server_version" gorm:"column:server_version; default:''; NOT NULL"`
	ApiServerAddress string        `json:"api_server_address" gorm:"column:api_server_address; default:''; NOT NULL"`
	Config           string        `json:"config" gorm:"column:config; default:''; NOT NULL;"`
	Addons           string        `json:"addons" gorm:"column:addons; default:''; NOT NULL;"`
	AddonsConfig     string        `json:"addons_config" gorm:"column:addons_config; default:''; NOT NULL;"`
	Status           ClusterStatus `json:"status" gorm:"column:status; default:0; NOT NULL;"`
	Type             ClusterType   `json:"type" gorm:"column:type; default:''; NOT NULL;"` //*  aws google cloud azure alicloud local
	KubeConfig       string        `json:"kube_config" gorm:"column:kube_config; default:''; NOT NULL; type:json"`
	PublicKey        string        `json:"public_key" gorm:"column:public_key; default:''; NOT NULL;"` // *
	Region           string        `json:"region" gorm:"column:region; default:''; NOT NULL;"`         // *
	VpcID            string        `json:"vpc_id" gorm:"column:vpc_id; default:''; NOT NULL;"`
	ExternalIP       string        `json:"external_ip" gorm:"column:external_ip; default:''; NOT NULL;"`
	AccessID         string        `json:"access_id" gorm:"column:access_id; default:''; NOT NULL;"`   // *
	AccessKey        string        `json:"access_key" gorm:"column:access_key; default:''; NOT NULL;"` // *
	BostionHost      *BostionHost  `json:"bostion_host" gorm:"-"`
	Logs             string        `json:"logs" gorm:"-"` // logs data from localfile
	Nodes            []*Node       `json:"nodes" gorm:"-"`
	NodeGroups       []*NodeGroup  `json:"node_groups" gorm:"-"`
	GPULabel         string        `json:"gpu_label" gorm:"column:gpu_label; default:''; NOT NULL;"`
	GPUTypes         string        `json:"gpu_types" gorm:"column:gpu_types; default:''; NOT NULL;"` // examlpe: 1080ti,2080ti,3090
	gorm.Model
}

type NodeGroup struct {
	ID                      int64         `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name                    string        `json:"name" gorm:"column:name; default:''; NOT NULL"`
	Type                    NodeGroupType `json:"type" gorm:"column:type; default:''; NOT NULL;"`
	InstanceType            string        `json:"instance_type" gorm:"column:instance_type; default:''; NOT NULL"`
	OSImage                 string        `json:"os_image" gorm:"column:os_image; default:''; NOT NULL"`
	CPU                     int32         `json:"cpu" gorm:"column:cpu; default:0; NOT NULL"`
	Memory                  float64       `json:"memory" gorm:"column:memory; default:0; NOT NULL"`
	GPU                     int32         `json:"gpu" gorm:"column:gpu; default:0; NOT NULL"`
	GpuSpec                 string        `json:"gpu_spec" gorm:"column:gpu_spec; default:''; NOT NULL"`      // NVIDIA Tesla V100（训练）、NVIDIA A100 Tensor Core、NVIDIA T4 GPU（推理）、NVIDIA A10G Tensor Core
	SystemDisk              int32         `json:"system_disk" gorm:"column:system_disk; default:0; NOT NULL"` // 随着服务释放掉的存储空间
	DataDisk                int32         `json:"data_disk" gorm:"column:data_disk; default:0; NOT NULL"`     // 挂载的存储空间
	InternetMaxBandwidthOut int32         `json:"internet_max_bandwidth_out" gorm:"column:internet_max_bandwidth_out; default:0; NOT NULL"`
	NodeInitScript          string        `json:"cloud_init_script" gorm:"column:cloud_init_script; default:''; NOT NULL"`
	MinSize                 int32         `json:"min_size" gorm:"column:min_size; default:0; NOT NULL"`
	MaxSize                 int32         `json:"max_size" gorm:"column:max_size; default:0; NOT NULL"`
	TargetSize              int32         `json:"target_size" gorm:"column:target_size; default:0; NOT NULL"`
	ClusterID               int64         `json:"cluster_id" gorm:"column:cluster_id; default:0; NOT NULL"`
}

type Node struct {
	ID          int64      `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	Name        string     `json:"name" gorm:"column:name; default:''; NOT NULL"`
	Labels      string     `json:"labels" gorm:"column:labels; default:''; NOT NULL"` // map[string]string json
	Kernel      string     `json:"kernel" gorm:"column:kernel; default:''; NOT NULL"`
	Container   string     `json:"container" gorm:"column:container; default:''; NOT NULL"`
	Kubelet     string     `json:"kubelet" gorm:"column:kubelet; default:''; NOT NULL"`
	KubeProxy   string     `json:"kube_proxy" gorm:"column:kube_proxy; default:''; NOT NULL"`
	InternalIP  string     `json:"internal_ip" gorm:"column:internal_ip; default:''; NOT NULL"`
	ExternalIP  string     `json:"external_ip" gorm:"column:external_ip; default:''; NOT NULL"`
	User        string     `json:"user" gorm:"column:user; default:''; NOT NULL"`
	Role        NodeRole   `json:"role" gorm:"column:role; default:''; NOT NULL;"` // master worker edge
	Status      NodeStatus `json:"status" gorm:"column:status; default:0; NOT NULL;"`
	ErrorInfo   string     `json:"error_info" gorm:"column:error_info; default:''; NOT NULL"`
	ClusterID   int64      `json:"cluster_id" gorm:"column:cluster_id; default:0; NOT NULL"`
	NodeGroup   *NodeGroup `json:"node_group" gorm:"-"`
	NodeGroupID int64      `json:"node_group_id" gorm:"column:node_group_id; default:0; NOT NULL"`
	NodePrice   float64    `json:"node_price" gorm:"column:node_price; default:0; NOT NULL;"` // 节点价格
	PodPrice    float64    `json:"pod_price" gorm:"column:pod_price; default:0; NOT NULL;"`   // 节点上pod的价格
	gorm.Model
}

type BostionHost struct {
	ID         int64  `json:"id" gorm:"column:id;primaryKey;AUTO_INCREMENT"`
	InstanceID string `json:"instance_id" gorm:"column:instance_id; default:''; NOT NULL"`
	Hostname   string `json:"hostname" gorm:"column:hostname; default:''; NOT NULL"`
	ExternalIP string `json:"external_ip" gorm:"column:external_ip; default:''; NOT NULL"`
	PublicIP   string `json:"public_ip" gorm:"column:public_ip; default:''; NOT NULL"`
	PrivateIP  string `json:"private_ip" gorm:"column:private_ip; default:''; NOT NULL"`
	ClusterID  int64  `json:"cluster_id" gorm:"column:cluster_id; default:0; NOT NULL"`
	CPU        int32  `json:"cpu" gorm:"column:cpu; default:0; NOT NULL"`
	Memory     int32  `json:"memory" gorm:"column:memory; default:0; NOT NULL"`
	gorm.Model
}

// 持久化
type ClusterRepo interface {
	Save(context.Context, *Cluster) error
	Get(context.Context, int64) (*Cluster, error)
	GetByName(context.Context, string) (*Cluster, error)
	List(context.Context, *Cluster) ([]*Cluster, error)
	Delete(context.Context, int64) error
	Put(ctx context.Context, cluster *Cluster) error
	Watch(ctx context.Context) (*Cluster, error)
}

// 基础建设
type Infrastructure interface {
	Start(context.Context, *Cluster) error
	Stop(context.Context, *Cluster) error
	Get(context.Context, *Cluster) error
}

// 集群配置
type ClusterConstruct interface {
	MigrateToBostionHost(context.Context, *Cluster) error
	Install(context.Context, *Cluster) error
	UnInstall(context.Context, *Cluster) error
	AddNodes(context.Context, *Cluster, []*Node) error
	RemoveNodes(context.Context, *Cluster, []*Node) error
	GenerateInitial(context.Context, *Cluster) error
	GenerateNodeLables(context.Context, *Cluster, *NodeGroup) (lables string, err error)
}

// 运行时集群
type ClusterRuntime interface {
	CurrentCluster(context.Context, *Cluster) error
}

var ErrClusterNotFound error = errors.New("cluster not found")
var ErrClusterApiServerAddressEmpty error = errors.New("cluster api server address is empty")
var ErrInfrastructureNotFound error = errors.New("infrastructure not found")

type ClusterType string

func (c ClusterType) String() string {
	return string(c)
}

const (
	ClusterTypeLocal    ClusterType = "local"
	ClusterTypeAWS      ClusterType = "aws"
	ClusterTypeGoogle   ClusterType = "google"
	ClusterTypeAzure    ClusterType = "azure"
	ClusterTypeAliCloud ClusterType = "alicloud"
)

type ClusterStatus uint8

func (s ClusterStatus) Uint8() uint8 {
	return uint8(s)
}

const (
	ClusterStatusUnspecified ClusterStatus = 0
	ClusterStatusRunning     ClusterStatus = 1
	ClusterStatusDeleted     ClusterStatus = 2
	ClusterStatucCreating    ClusterStatus = 3
)

var (
	ClusterStatusName = map[uint8]string{
		0: "unspecified",
		1: "running",
		2: "deleted",
		3: "creating",
	}
	ClusterStatusValue = map[string]uint8{
		"unspecified": 0,
		"running":     1,
		"deleted":     2,
		"creating":    3,
	}
)

type Nodes []*Node

func (n Nodes) Len() int {
	return len(n)
}

func (n Nodes) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

// 从小到大排序
func (n Nodes) Less(i, j int) bool {
	if n[i].NodeGroup == nil || n[j].NodeGroup == nil {
		return false
	}
	if n[i].NodeGroup.Memory == n[j].NodeGroup.Memory {
		return n[i].NodeGroup.CPU < n[j].NodeGroup.CPU
	}
	return n[i].NodeGroup.Memory < n[j].NodeGroup.Memory
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

type NodeSize int32

func (n NodeSize) Int32() int32 {
	return int32(n)
}

func (n NodeSize) Int() int {
	return int(n)
}

const (
	NodeMinSize NodeSize = 5
)

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
	// an Unspecified instanceState means the actual instance status is undefined (nil).
	NodeStatusUnspecified NodeStatus = 0
	// NodeStatusRunning means instance is running.
	NodeStatusRunning NodeStatus = 1
	// NodeStatusCreating means instance is being created.
	NodeStatusCreating NodeStatus = 2
	// NodeStatusDeleting means instance is being deleted.
	NodeStatusDeleting NodeStatus = 3
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

func (c *Cluster) GetNode(nodeId int64) *Node {
	for _, node := range c.Nodes {
		if node.ID == nodeId {
			return node
		}
	}
	return nil
}

func (ng *NodeGroup) SetTargetSize(size int32) {
	ng.TargetSize = size
}

func (c *Cluster) GetType() ClusterType {
	return ClusterType(c.Type)
}

func (n *Node) GetStatus() NodeStatus {
	return NodeStatus(n.Status)
}

type ClusterUsecase struct {
	clusterRepo      ClusterRepo
	infrastructure   Infrastructure
	clusterConstruct ClusterConstruct
	clusterRuntime   ClusterRuntime
	log              *log.Helper
}

func NewClusterUseCase(clusterRepo ClusterRepo, infrastructure Infrastructure, clusterConstruct ClusterConstruct, clusterRuntime ClusterRuntime, logger log.Logger) *ClusterUsecase {
	c := &ClusterUsecase{
		clusterRepo:      clusterRepo,
		infrastructure:   infrastructure,
		clusterConstruct: clusterConstruct,
		clusterRuntime:   clusterRuntime,
		log:              log.NewHelper(logger),
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
	if !isClusterEmpty(cluster) && cluster.ID != data.ID {
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

// 获取当前集群最新信息
func (uc *ClusterUsecase) GetCurrentCluster(ctx context.Context) (*Cluster, error) {
	cluster := &Cluster{}
	err := uc.clusterRuntime.CurrentCluster(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

// 根据nodegroup增加节点
func (uc *ClusterUsecase) NodeGroupIncreaseSize(ctx context.Context, cluster *Cluster, nodeGroup *NodeGroup, size int32) error {
	for i := 0; i < int(size); i++ {
		node := &Node{
			Name:        fmt.Sprintf("%s-%s", cluster.Name, utils.GetRandomString()),
			Role:        NodeRoleWorker,
			Status:      NodeStatusCreating,
			ClusterID:   cluster.ID,
			NodeGroupID: nodeGroup.ID,
		}
		cluster.Nodes = append(cluster.Nodes, node)
	}
	return uc.Apply(ctx, cluster)
}

// 删除节点
func (uc *ClusterUsecase) DeleteNodes(ctx context.Context, cluster *Cluster, nodes []*Node) error {
	for _, node := range nodes {
		for i, n := range cluster.Nodes {
			if n.ID == node.ID {
				cluster.Nodes = append(cluster.Nodes[:i], cluster.Nodes[i+1:]...)
				break
			}
		}
	}
	return uc.Apply(ctx, cluster)
}

// 预测一个节点配置，也就是根据当前节点组目前还可以配置的节点
func (uc *ClusterUsecase) NodeGroupTemplateNodeInfo(ctx context.Context, cluster *Cluster, nodeGroup *NodeGroup) (*Node, error) {
	nodeLables, err := uc.clusterConstruct.GenerateNodeLables(ctx, cluster, nodeGroup)
	if err != nil {
		return nil, err
	}
	return &Node{
		Name:        fmt.Sprintf("%s-%s", cluster.Name, utils.GetRandomString()),
		Role:        NodeRoleWorker,
		Status:      NodeStatusCreating,
		ClusterID:   cluster.ID,
		NodeGroupID: nodeGroup.ID,
		Labels:      nodeLables,
	}, nil
}

// 在云提供商销毁前清理打开的资源，例如协程等
func (uc *ClusterUsecase) Cleanup(ctx context.Context) error {
	return nil
}

// 在每个主循环前调用，用于动态更新云提供商状态
func (uc *ClusterUsecase) Refresh(ctx context.Context) error {
	// 获取当前集群状态更新状态
	cluster := &Cluster{}
	err := uc.clusterRuntime.CurrentCluster(ctx, cluster)
	if err != nil {
		return err
	}
	cluster, err = uc.clusterRepo.GetByName(ctx, cluster.Name)
	if err != nil {
		return err
	}
	for _, v := range cluster.Nodes {
		for _, currentNode := range cluster.Nodes {
			if v.Name == currentNode.Name {
				v.Status = currentNode.Status
				break
			}
		}
	}
	err = uc.Save(ctx, cluster)
	if err != nil {
		return err
	}
	return nil
}

func (uc *ClusterUsecase) Apply(ctx context.Context, cluster *Cluster) (err error) {
	if ClusterStatus(cluster.Status) == ClusterStatucCreating {
		err = uc.clusterConstruct.GenerateInitial(ctx, cluster)
		if err != nil {
			return err
		}
	}
	err = uc.clusterRepo.Put(ctx, cluster)
	if err != nil {
		return err
	}
	return nil
}

func (uc *ClusterUsecase) Watch(ctx context.Context) (*Cluster, error) {
	return uc.clusterRepo.Watch(ctx)
}

func (uc *ClusterUsecase) Reconcile(ctx context.Context, cluster *Cluster) (err error) {
	if cluster.IsDeleteed() {
		err = uc.clusterConstruct.UnInstall(ctx, cluster)
		if err != nil {
			return err
		}
		err = uc.infrastructure.Stop(ctx, cluster)
		if err != nil {
			return err
		}
		return nil
	}
	err = uc.infrastructure.Get(ctx, cluster)
	if errors.Is(err, ErrInfrastructureNotFound) {
		ctx, cancel := context.WithCancel(ctx)
		defer func() {
			if err == nil {
				cancel()
			}
		}()
		err = uc.infrastructure.Start(ctx, cluster)
		if err != nil {
			return err
		}
		err = uc.clusterRepo.Save(ctx, cluster)
		if err != nil {
			return err
		}
		err = uc.clusterConstruct.MigrateToBostionHost(ctx, cluster)
		if err != nil {
			return err
		}
		return nil
	}
	if err != nil {
		return err
	}
	currentCluster := new(Cluster)
	err = uc.clusterRuntime.CurrentCluster(ctx, currentCluster)
	if errors.Is(err, ErrClusterNotFound) {
		err = uc.clusterConstruct.Install(ctx, cluster)
		if err != nil {
			return err
		}
		return nil
	}
	if err != nil {
		return err
	}
	err = uc.infrastructure.Start(ctx, cluster)
	if err != nil {
		return err
	}
	err = uc.handlerAddNode(ctx, cluster, currentCluster)
	if err != nil {
		return err
	}
	err = uc.handlerRemoveNode(ctx, cluster, currentCluster)
	if err != nil {
		return err
	}
	return nil
}

func (uc *ClusterUsecase) handlerAddNode(ctx context.Context, cluster, runingCluster *Cluster) error {
	addNodes := make([]*Node, 0)
	for _, node := range cluster.Nodes {
		nodeExist := false
		for _, currentNode := range runingCluster.Nodes {
			if node.Name == currentNode.Name {
				nodeExist = true
				break
			}
		}
		if !nodeExist {
			addNodes = append(addNodes, node)
		}
	}
	err := uc.clusterConstruct.AddNodes(ctx, runingCluster, addNodes)
	if err != nil {
		return err
	}
	return nil
}

func (uc *ClusterUsecase) handlerRemoveNode(ctx context.Context, cluster, runingCluster *Cluster) error {
	removeNodes := make([]*Node, 0)
	for _, currentNode := range runingCluster.Nodes {
		nodeExist := false
		for _, node := range cluster.Nodes {
			if currentNode.Name == node.Name {
				nodeExist = true
				break
			}
		}
		if !nodeExist {
			removeNodes = append(removeNodes, currentNode)
		}
	}
	err := uc.clusterConstruct.RemoveNodes(ctx, runingCluster, removeNodes)
	if err != nil {
		return err
	}
	return nil
}

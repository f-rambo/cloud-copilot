package biz

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/f-rambo/cloud-copilot/utils"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
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

func (c *Cluster) AddSubCloudResource(resourceType ResourceType, parentID string, resource *CloudResource) {
	cloudResource := c.GetCloudResourceByID(resourceType, parentID)
	if cloudResource == nil {
		return
	}
	if cloudResource.SubResources == nil {
		cloudResource.SubResources = make([]*CloudResource, 0)
	}
	resource.Type = resourceType
	if resource.Id == "" {
		resource.Id = uuid.New().String()
	}
	cloudResource.SubResources = append(cloudResource.SubResources, resource)
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
	resource := getCloudResourceByID(c.GetCloudResource(resourceType), id)
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

func getCloudResourceByID(cloudResources []*CloudResource, id string) *CloudResource {
	for _, resource := range cloudResources {
		if resource.Id == id {
			return resource
		}
		if len(resource.SubResources) > 0 {
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
		if resource.Tags == "" {
			continue
		}
		resourceTagsMap := c.DecodeTags(resource.Tags)
		match := true
		for i := 0; i < len(tagKeyValues); i += 2 {
			tagKey := tagKeyValues[i]
			tagValue := tagKeyValues[i+1]
			if resourceTagsMap[tagKey] != tagValue {
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

func (c *Cluster) EncodeTags(tags map[string]string) string {
	tagStr := ""
	for key, value := range tags {
		tagStr += fmt.Sprintf("%s:%s,", key, value)
	}
	return tagStr
}

func (c *Cluster) DecodeTags(tags string) map[string]string {
	tagsMap := make(map[string]string)
	for _, tag := range strings.Split(tags, ",") {
		tagKeyValue := strings.Split(tag, ":")
		if len(tagKeyValue) != 2 {
			continue
		}
		tagsMap[tagKeyValue[0]] = tagKeyValue[1]
	}
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
		if resources.Type == resourceType && resources.Id != id {
			cloudResources = append(cloudResources, resources)
		}
	}
	c.CloudResources = cloudResources
}

// delete cloud resource by resourceType and tag value and tag key
func (c *Cluster) DeleteCloudResourceByTags(resourceType ResourceType, tagKeyValues ...string) {
	cloudResources := make([]*CloudResource, 0)
	for _, resource := range c.CloudResources {
		if resource.Tags == "" {
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

func (c *Cluster) GenerateNodeGroupName(nodeGroup *NodeGroup) {
	nodeGroup.Name = strings.Join([]string{
		c.Name,
		nodeGroup.Type.String(),
		nodeGroup.Os,
		nodeGroup.Arch,
		cast.ToString(nodeGroup.Cpu),
		cast.ToString(nodeGroup.Memory),
		cast.ToString(nodeGroup.Gpu),
		cast.ToString(nodeGroup.GpuSpec),
	}, "-")
}

func (c ClusterType) IsCloud() bool {
	return c != ClusterType_LOCAL
}

func (c ClusterType) IsIntegratedCloud() bool {
	return c == ClusterType_AWS_EKS || c == ClusterType_ALICLOUD_AKS
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
		return n[i].Cpu < n[j].Cpu
	}
	return n[i].Memory < n[j].Memory
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

func (uc *ClusterUsecase) Get(ctx context.Context, id int64) (*Cluster, error) {
	cluster, err := uc.clusterData.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (uc *ClusterUsecase) List(ctx context.Context) ([]*Cluster, error) {
	return uc.clusterData.List(ctx, nil)
}

func (uc *ClusterUsecase) Delete(ctx context.Context, clusterID int64) error {
	cluster, err := uc.clusterData.Get(ctx, clusterID)
	if err != nil {
		return err
	}
	if cluster == nil {
		return nil
	}
	for _, node := range cluster.Nodes {
		node.Status = NodeStatus_NODE_DELETING
	}
	err = uc.clusterData.Save(ctx, cluster)
	if err != nil {
		return err
	}
	err = uc.clusterData.Delete(ctx, clusterID)
	if err != nil {
		return err
	}
	return nil
}

func (uc *ClusterUsecase) Save(ctx context.Context, cluster *Cluster) error {
	data, err := uc.clusterData.GetByName(ctx, cluster.Name)
	if err != nil {
		return err
	}
	if data != nil && cluster.Id != data.Id {
		return errors.New("cluster name already exists")
	}
	if cluster.Level.String() == "" {
		cluster.Level = ClusterLevel_BASIC
	}
	err = uc.clusterData.Save(ctx, cluster)
	if err != nil {
		return err
	}
	uc.apply(cluster)
	return nil
}

func (uc *ClusterUsecase) GetRegions(ctx context.Context, cluster *Cluster) ([]string, error) {
	if cluster.Type == ClusterType_LOCAL {
		return []string{}, nil
	}
	err := uc.clusterInfrastructure.GetRegions(ctx, cluster)
	if err != nil {
		return nil, err
	}
	regionNames := make([]string, 0)
	for _, region := range cluster.GetCloudResource(ResourceType_AVAILABILITY_ZONES) {
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
			return nil
		}
	}
}

func (uc *ClusterUsecase) Stop(ctx context.Context) error {
	close(uc.eventChan)
	return nil
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
	if cluster.DeletedAt != nil {
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
	nodegroup := &NodeGroup{Id: uuid.New().String(), ClusterId: c.Id}
	nodegroup.Type = NodeGroupType_NORMAL
	c.GenerateNodeGroupName(nodegroup)
	nodegroup.Cpu = 2
	nodegroup.Memory = 4
	nodegroup.TargetSize = 3
	nodegroup.MinSize = 1
	nodegroup.MaxSize = 5
	if c.Level != ClusterLevel_BASIC {
		nodegroup.Cpu = 4
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
			Status:      NodeStatus_NODE_CREATING,
			ClusterId:   c.Id,
			NodeGroupId: nodegroup.Id,
		}
		if i < 3 {
			node.Role = NodeRole_MASTER
		} else {
			node.Role = NodeRole_WORKER
		}
		node.Labels = labels
		c.Nodes = append(c.Nodes, node)
	}
	c.BostionHost = &BostionHost{
		Id:        uuid.New().String(),
		ClusterId: c.Id,
		Hostname:  "bostion-host",
		Status:    NodeStatus_NODE_CREATING,
		Cpu:       2,
		Memory:    4,
		SshPort:   22,
	}
}

func (uc *ClusterUsecase) handlerClusterNotInstalled(ctx context.Context, cluster *Cluster) error {
	cluster.SettingSpecifications()
	err := uc.clusterInfrastructure.GetRegions(ctx, cluster)
	if err != nil {
		return err
	}
	if cluster.Level == ClusterLevel_BASIC {
		singleZone := cluster.GetSingleCloudResource(ResourceType_AVAILABILITY_ZONES)
		cluster.DeleteCloudResource(ResourceType_AVAILABILITY_ZONES)
		cluster.AddCloudResource(singleZone)
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

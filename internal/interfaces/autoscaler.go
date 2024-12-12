package interfaces

import (
	"context"
	"encoding/json"

	autoscaler "github.com/f-rambo/cloud-copilot/api/autoscaler"
	"github.com/f-rambo/cloud-copilot/internal/biz"
	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	anypb "google.golang.org/protobuf/types/known/anypb"
	corev1 "k8s.io/api/core/v1"
)

// clusterautoscaler.cloudprovider.v1.externalgrpc.CloudProvider

type Autoscaler struct {
	autoscaler.UnimplementedCloudProviderServer
	clusterUc *biz.ClusterUsecase
	c         *conf.Bootstrap
	log       *log.Helper
}

func NewAutoscaler(clusterUc *biz.ClusterUsecase, c *conf.Bootstrap, logger log.Logger) *Autoscaler {
	return &Autoscaler{
		clusterUc: clusterUc,
		c:         c,
		log:       log.NewHelper(logger),
	}
}

func pdNodeGroup(nodeGroup *biz.NodeGroup) *autoscaler.NodeGroup {
	return &autoscaler.NodeGroup{
		Id:      cast.ToString(nodeGroup.Id),
		MinSize: nodeGroup.MinSize,
		MaxSize: nodeGroup.MaxSize,
		Debug:   nodeGroup.Type.String(),
	}
}

func getNodeByName(nodeName string, cluster *biz.Cluster) *biz.Node {
	for _, node := range cluster.Nodes {
		if node.Name == nodeName {
			return node
		}
	}
	return nil
}

func getNodeGroupByID(nodeGroupID string, cluster *biz.Cluster) *biz.NodeGroup {
	for _, nodeGroup := range cluster.NodeGroups {
		if cast.ToString(nodeGroup.Id) == nodeGroupID {
			return nodeGroup
		}
	}
	return nil
}

func getNodesByNGIDAndName(nodeGroupID string, nodeName string, cluster *biz.Cluster) *biz.Node {
	nodeGroup := getNodeGroupByID(nodeGroupID, cluster)
	if nodeGroup == nil {
		return nil
	}
	for _, node := range cluster.Nodes {
		nodeGroup := cluster.GetNodeGroup(node.NodeGroupId)
		if nodeGroup == nil {
			continue
		}
		if node.Name == nodeName && cast.ToString(nodeGroup.Id) == nodeGroupID {
			return node
		}
	}
	return nil
}

func nodeToV1Node(node *biz.Node) *corev1.Node {
	coreNode := &corev1.Node{}
	coreNode.Kind = "Node"
	coreNode.APIVersion = "v1"
	coreNode.Name = node.Name
	lables := make(map[string]string)
	json.Unmarshal([]byte(node.Labels), &lables)
	coreNode.Labels = lables
	return coreNode
}

// NodeGroups：返回配置的所有节点组。
// NodeGroups returns all node groups configured for this cloud provider.
func (a *Autoscaler) NodeGroups(ctx context.Context, in *autoscaler.NodeGroupsRequest) (*autoscaler.NodeGroupsResponse, error) {
	a.log.Infof("NodeGroups got gRPC request: %T %s", in, in)
	cluster, err := a.clusterUc.GetCurrentCluster(ctx)
	if err != nil {
		return nil, err
	}
	responses := &autoscaler.NodeGroupsResponse{
		NodeGroups: make([]*autoscaler.NodeGroup, 0),
	}
	for _, nodeGroup := range cluster.NodeGroups {
		responses.NodeGroups = append(responses.NodeGroups, pdNodeGroup(nodeGroup))
	}
	return responses, nil
}

// NodeGroupForNode：返回给定节点所属的节点组。
// NodeGroupForNode returns the node group for the given node.
// The node group id is an empty string if the node should not
// be processed by cluster autoscaler.
func (a *Autoscaler) NodeGroupForNode(ctx context.Context, in *autoscaler.NodeGroupForNodeRequest) (*autoscaler.NodeGroupForNodeResponse, error) {
	a.log.Infof("NodeGroupForNode got gRPC request: %T %s", in, in)
	node := in.GetNode()
	if node == nil {
		return nil, errors.New("req node is nil")
	}
	cluster, err := a.clusterUc.GetCurrentCluster(ctx)
	if err != nil {
		return nil, err
	}
	resNode := getNodeByName(node.Name, cluster)
	if resNode == nil {
		return nil, errors.New("node not found")
	}
	nodeGroup := cluster.GetNodeGroup(resNode.NodeGroupId)
	if nodeGroup == nil {
		return nil, errors.New("node group not found")
	}
	return &autoscaler.NodeGroupForNodeResponse{
		NodeGroup: pdNodeGroup(nodeGroup),
	}, nil
}

// PricingNodePrice：返回在指定时间段内运行一个节点的理论最低价格。
// PricingNodePrice returns a theoretical minimum price of running a node for
// a given period of time on a perfectly matching machine.
// Implementation optional: if unimplemented return error code 12 (for `Unimplemented`)
func (a *Autoscaler) PricingNodePrice(ctx context.Context, in *autoscaler.PricingNodePriceRequest) (*autoscaler.PricingNodePriceResponse, error) {
	a.log.Infof("PricingNodePrice got gRPC request: %T %s", in, in)
	node := in.GetNode()
	cluster, err := a.clusterUc.GetCurrentCluster(ctx)
	if err != nil {
		return nil, err
	}
	resNode := getNodeByName(node.Name, cluster)
	if resNode == nil {
		return nil, errors.New("node not found")
	}
	nodeGroup := cluster.GetNodeGroup(resNode.NodeGroupId)
	return &autoscaler.PricingNodePriceResponse{Price: float64(nodeGroup.NodePrice)}, nil
}

// PricingPodPrice：返回在指定时间段内运行一个 Pod 的理论最低价格。
// PricingPodPrice returns a theoretical minimum price of running a pod for a given
// period of time on a perfectly matching machine.
// Implementation optional: if unimplemented return error code 12 (for `Unimplemented`)
func (a *Autoscaler) PricingPodPrice(ctx context.Context, in *autoscaler.PricingPodPriceRequest) (*autoscaler.PricingPodPriceResponse, error) {
	a.log.Infof("PricingPodPrice got gRPC request: %T %s", in, in)
	pod := in.GetPod()
	cluster, err := a.clusterUc.GetCurrentCluster(ctx)
	if err != nil {
		return nil, err
	}
	resNode := getNodeByName(pod.Spec.NodeName, cluster)
	if resNode == nil {
		return nil, errors.New("node not found")
	}
	nodeGroup := cluster.GetNodeGroup(resNode.NodeGroupId)
	return &autoscaler.PricingPodPriceResponse{Price: float64(nodeGroup.NodePrice)}, nil
}

// GPULabel：返回添加到具有 GPU 资源的节点的标签。
// GPULabel returns the label added to nodes with GPU resource.
func (a *Autoscaler) GPULabel(ctx context.Context, in *autoscaler.GPULabelRequest) (*autoscaler.GPULabelResponse, error) {
	a.log.Infof("GPULabel got gRPC request: %T %s", in, in)
	cluster, err := a.clusterUc.GetCurrentCluster(ctx)
	if err != nil {
		return nil, err
	}
	gpuLable := ""
	for _, node := range cluster.Nodes {
		nodeGroup := cluster.GetNodeGroup(node.NodeGroupId)
		if nodeGroup == nil {
			continue
		}
		if nodeGroup.GpuSpec != biz.NodeGPUSpec_NodeGPUSpec_UNSPECIFIED {
			gpuLable = nodeGroup.GpuSpec.String()
			break
		}
	}
	return &autoscaler.GPULabelResponse{Label: gpuLable}, nil
}

// GetAvailableGPUTypes：返回云提供商支持的所有 GPU 类型。
// GetAvailableGPUTypes return all available GPU types cloud provider supports.
func (a *Autoscaler) GetAvailableGPUTypes(ctx context.Context, in *autoscaler.GetAvailableGPUTypesRequest) (*autoscaler.GetAvailableGPUTypesResponse, error) {
	a.log.Infof("GetAvailableGPUTypes got gRPC request: %T %s", in, in)
	cluster, err := a.clusterUc.GetCurrentCluster(ctx)
	if err != nil {
		return nil, err
	}
	pbGpuTypes := make(map[string]*anypb.Any)
	for _, node := range cluster.Nodes {
		nodeGroup := cluster.GetNodeGroup(node.NodeGroupId)
		if nodeGroup == nil {
			continue
		}
		if nodeGroup.GpuSpec == biz.NodeGPUSpec_NodeGPUSpec_UNSPECIFIED {
			continue
		}
		pbGpuTypes[nodeGroup.GpuSpec.String()] = nil
	}
	return &autoscaler.GetAvailableGPUTypesResponse{
		GpuTypes: pbGpuTypes,
	}, nil
}

// Cleanup：在云提供商销毁前清理打开的资源，例如协程等。
// Cleanup cleans up open resources before the cloud provider is destroyed, i.e. go routines etc.
func (a *Autoscaler) Cleanup(ctx context.Context, in *autoscaler.CleanupRequest) (*autoscaler.CleanupResponse, error) {
	a.log.Infof("Cleanup got gRPC request: %T %s", in, in)
	err := a.clusterUc.Cleanup(ctx)
	if err != nil {
		return nil, err
	}
	return &autoscaler.CleanupResponse{}, nil
}

// Refresh：在每个主循环前调用，用于动态更新云提供商状态。
// Refresh is called before every main loop and can be used to dynamically update cloud provider state.
func (a *Autoscaler) Refresh(ctx context.Context, in *autoscaler.RefreshRequest) (*autoscaler.RefreshResponse, error) {
	a.log.Infof("Refresh got gRPC request: %T %s", in, in)
	err := a.clusterUc.Refresh(ctx)
	if err != nil {
		return nil, err
	}
	return &autoscaler.RefreshResponse{}, nil
}

// NodeGroupTargetSize：返回节点组的当前目标大小。
// NodeGroupTargetSize returns the current target size of the node group. It is possible
// that the number of nodes in Kubernetes is different at the moment but should be equal
// to the size of a node group once everything stabilizes (new nodes finish startup and
// registration or removed nodes are deleted completely).
func (a *Autoscaler) NodeGroupTargetSize(ctx context.Context, in *autoscaler.NodeGroupTargetSizeRequest) (*autoscaler.NodeGroupTargetSizeResponse, error) {
	a.log.Infof("NodeGroupTargetSize got gRPC request: %T %s", in, in)
	nodeGroupId := in.GetId()
	cluster, err := a.clusterUc.GetCurrentCluster(ctx)
	if err != nil {
		return nil, err
	}
	resNodegroup := getNodeGroupByID(nodeGroupId, cluster)
	if resNodegroup == nil {
		return nil, errors.New("node group not found")
	}
	return &autoscaler.NodeGroupTargetSizeResponse{TargetSize: resNodegroup.TargetSize}, nil
}

// NodeGroupIncreaseSize：增加节点组的大小。
// NodeGroupIncreaseSize increases the size of the node group. To delete a node you need
// to explicitly name it and use NodeGroupDeleteNodes. This function should wait until
// node group size is updated.
func (a *Autoscaler) NodeGroupIncreaseSize(ctx context.Context, in *autoscaler.NodeGroupIncreaseSizeRequest) (*autoscaler.NodeGroupIncreaseSizeResponse, error) {
	a.log.Infof("NodeGroupIncreaseSize got gRPC request: %T %s", in, in)
	nodeGroupId := in.GetId()
	delta := in.GetDelta()
	cluster, err := a.clusterUc.GetCurrentCluster(ctx)
	if err != nil {
		return nil, err
	}
	resNodegroup := getNodeGroupByID(nodeGroupId, cluster)
	if resNodegroup == nil {
		return nil, errors.New("node group not found")
	}
	err = a.clusterUc.NodeGroupIncreaseSize(ctx, cluster, resNodegroup, delta)
	if err != nil {
		return nil, err
	}
	return &autoscaler.NodeGroupIncreaseSizeResponse{}, nil
}

// NodeGroupDeleteNodes：从节点组中删除节点，同时减少节点组的大小。
// NodeGroupDeleteNodes deletes nodes from this node group (and also decreasing the size
// of the node group with that). Error is returned either on failure or if the given node
// doesn't belong to this node group. This function should wait until node group size is updated.
func (a *Autoscaler) NodeGroupDeleteNodes(ctx context.Context, in *autoscaler.NodeGroupDeleteNodesRequest) (*autoscaler.NodeGroupDeleteNodesResponse, error) {
	a.log.Infof("NodeGroupDeleteNodes got gRPC request: %T %s", in, in)
	nodeGroupId := in.GetId()
	cluster, err := a.clusterUc.GetCurrentCluster(ctx)
	if err != nil {
		return nil, err
	}
	resNodeGroup := getNodeGroupByID(nodeGroupId, cluster)
	if resNodeGroup == nil {
		return nil, errors.New("node group not found")
	}
	nodes := make([]*biz.Node, 0)
	for _, node := range in.GetNodes() {
		resNode := getNodesByNGIDAndName(nodeGroupId, node.Name, cluster)
		if resNode == nil {
			return nil, errors.New("node not found")
		}
		nodes = append(nodes, resNode)
	}
	err = a.clusterUc.DeleteNodes(ctx, cluster, nodes)
	if err != nil {
		return nil, err
	}
	return &autoscaler.NodeGroupDeleteNodesResponse{}, nil
}

// NodeGroupDecreaseTargetSize：减少节点组的目标大小。
// NodeGroupDecreaseTargetSize decreases the target size of the node group. This function
// doesn't permit to delete any existing node and can be used only to reduce the request
// for new nodes that have not been yet fulfilled. Delta should be negative. It is assumed
// that cloud provider will not delete the existing nodes if the size when there is an option
// to just decrease the target.
func (a *Autoscaler) NodeGroupDecreaseTargetSize(ctx context.Context, in *autoscaler.NodeGroupDecreaseTargetSizeRequest) (*autoscaler.NodeGroupDecreaseTargetSizeResponse, error) {
	a.log.Infof("NodeGroupDecreaseTargetSize got gRPC request: %T %s", in, in)
	nodeGroupId := in.GetId()
	delta := in.GetDelta()
	if delta >= 0 {
		return nil, errors.New("delta must be negative")
	}
	cluster, err := a.clusterUc.GetCurrentCluster(ctx)
	if err != nil {
		return nil, err
	}
	resNodegroup := getNodeGroupByID(nodeGroupId, cluster)
	if resNodegroup == nil {
		return nil, errors.New("node group not found")
	}
	newSize := resNodegroup.TargetSize + delta
	if newSize < 0 {
		return nil, errors.New("new target size must be non-negative")
	}
	resNodegroup.SetTargetSize(newSize)
	err = a.clusterUc.Save(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return &autoscaler.NodeGroupDecreaseTargetSizeResponse{}, nil
}

// NodeGroupNodes：返回属于该节点组的所有节点列表。
// NodeGroupNodes returns a list of all nodes that belong to this node group.
func (a *Autoscaler) NodeGroupNodes(ctx context.Context, in *autoscaler.NodeGroupNodesRequest) (*autoscaler.NodeGroupNodesResponse, error) {
	a.log.Infof("NodeGroupNodes got gRPC request: %T %s", in, in)
	nodeGroupId := in.GetId()
	cluster, err := a.clusterUc.GetCurrentCluster(ctx)
	if err != nil {
		return nil, err
	}
	resNodegroup := getNodeGroupByID(nodeGroupId, cluster)
	if resNodegroup == nil {
		return nil, errors.New("node group not found")
	}
	nodes := make([]*biz.Node, 0)
	for _, node := range cluster.Nodes {
		if cast.ToString(node.NodeGroupId) == nodeGroupId {
			nodes = append(nodes, node)
		}
	}
	instances := make([]*autoscaler.Instance, 0)
	for _, node := range nodes {
		instance := new(autoscaler.Instance)
		instance.Id = cast.ToString(node.Id)
		if node.Status == biz.NodeStatus_NODE_UNSPECIFIED {
			instance.Status = &autoscaler.InstanceStatus{
				InstanceState: autoscaler.InstanceStatus_unspecified,
				ErrorInfo:     &autoscaler.InstanceErrorInfo{},
			}
		} else {
			instance.Status = new(autoscaler.InstanceStatus)
			instance.Status.InstanceState = autoscaler.InstanceStatus_InstanceState(node.Status)
			if node.ErrorInfo == "" {
				instance.Status.ErrorInfo = &autoscaler.InstanceErrorInfo{}
			} else {
				instance.Status.ErrorInfo = &autoscaler.InstanceErrorInfo{
					ErrorCode:    string(node.Status),
					ErrorMessage: node.ErrorInfo,
				}
			}
		}
		instances = append(instances, instance)
	}
	return &autoscaler.NodeGroupNodesResponse{Instances: instances}, nil
}

// NodeGroupTemplateNodeInfo：返回一个空节点的结构，用于扩展模拟。
// NodeGroupTemplateNodeInfo returns a structure of an empty (as if just started) node,
// with all of the labels, capacity and allocatable information. This will be used in
// scale-up simulations to predict what would a new node look like if a node group was expanded.
// Implementation optional: if unimplemented return error code 12 (for `Unimplemented`)
func (a *Autoscaler) NodeGroupTemplateNodeInfo(ctx context.Context, in *autoscaler.NodeGroupTemplateNodeInfoRequest) (*autoscaler.NodeGroupTemplateNodeInfoResponse, error) {
	a.log.Infof("NodeGroupTemplateNodeInfo got gRPC request: %T %s", in, in)
	nodeGroupID := in.GetId()
	cluster, err := a.clusterUc.GetCurrentCluster(ctx)
	if err != nil {
		return nil, err
	}
	resNodegroup := getNodeGroupByID(nodeGroupID, cluster)
	if resNodegroup == nil {
		return nil, errors.New("node group not found")
	}
	node, err := a.clusterUc.NodeGroupTemplateNodeInfo(ctx, cluster, resNodegroup)
	if err != nil {
		return nil, status.Error(codes.Unimplemented, err.Error())
	}
	return &autoscaler.NodeGroupTemplateNodeInfoResponse{NodeInfo: nodeToV1Node(node)}, nil
}

// NodeGroupGetOptions：返回该节点组应使用的自动扩展选项。
// GetOptions returns NodeGroupAutoscalingOptions that should be used for this particular
// NodeGroup.
// Implementation optional: if unimplemented return error code 12 (for `Unimplemented`)
func (a *Autoscaler) NodeGroupGetOptions(ctx context.Context, in *autoscaler.NodeGroupAutoscalingOptionsRequest) (*autoscaler.NodeGroupAutoscalingOptionsResponse, error) {
	a.log.Infof("NodeGroupGetOptions got gRPC request: %T %s", in, in)
	defaultopts := in.GetDefaults()
	// todo
	return &autoscaler.NodeGroupAutoscalingOptionsResponse{NodeGroupAutoscalingOptions: defaultopts}, nil
}

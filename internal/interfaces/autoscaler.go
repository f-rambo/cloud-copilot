package interfaces

import (
	"context"

	"github.com/f-rambo/ocean/api/autoscaler"
	"github.com/f-rambo/ocean/internal/biz"
	"github.com/f-rambo/ocean/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	grpc "google.golang.org/grpc"
)

// examples : https://github.com/kubernetes/autoscaler/tree/master/cluster-autoscaler/cloudprovider/externalgrpc/examples/external-grpc-cloud-provider-service

type Autoscaler struct {
	autoscaler.UnimplementedAutoscalerServiceServer
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

// NodeGroups returns all node groups configured for this cloud provider.
func (a *Autoscaler) NodeGroups(ctx context.Context, in *autoscaler.NodeGroupsRequest, opts ...grpc.CallOption) (*autoscaler.NodeGroupsResponse, error) {
	return nil, nil
}

// NodeGroupForNode returns the node group for the given node.
// The node group id is an empty string if the node should not
// be processed by cluster autoscaler.
func (a *Autoscaler) NodeGroupForNode(ctx context.Context, in *autoscaler.NodeGroupForNodeRequest, opts ...grpc.CallOption) (*autoscaler.NodeGroupForNodeResponse, error) {
	return nil, nil
}

// PricingNodePrice returns a theoretical minimum price of running a node for
// a given period of time on a perfectly matching machine.
// Implementation optional: if unimplemented return error code 12 (for `Unimplemented`)
func (a *Autoscaler) PricingNodePrice(ctx context.Context, in *autoscaler.PricingNodePriceRequest, opts ...grpc.CallOption) (*autoscaler.PricingNodePriceResponse, error) {
	return nil, nil
}

// PricingPodPrice returns a theoretical minimum price of running a pod for a given
// period of time on a perfectly matching machine.
// Implementation optional: if unimplemented return error code 12 (for `Unimplemented`)
func (a *Autoscaler) PricingPodPrice(ctx context.Context, in *autoscaler.PricingPodPriceRequest, opts ...grpc.CallOption) (*autoscaler.PricingPodPriceResponse, error) {
	return nil, nil
}

// GPULabel returns the label added to nodes with GPU resource.
func (a *Autoscaler) GPULabel(ctx context.Context, in *autoscaler.GPULabelRequest, opts ...grpc.CallOption) (*autoscaler.GPULabelResponse, error) {
	return nil, nil
}

// GetAvailableGPUTypes return all available GPU types cloud provider supports.
func (a *Autoscaler) GetAvailableGPUTypes(ctx context.Context, in *autoscaler.GetAvailableGPUTypesRequest, opts ...grpc.CallOption) (*autoscaler.GetAvailableGPUTypesResponse, error) {
	return nil, nil
}

// Cleanup cleans up open resources before the cloud provider is destroyed, i.e. go routines etc.
func (a *Autoscaler) Cleanup(ctx context.Context, in *autoscaler.CleanupRequest, opts ...grpc.CallOption) (*autoscaler.CleanupResponse, error) {
	return nil, nil
}

// Refresh is called before every main loop and can be used to dynamically update cloud provider state.
func (a *Autoscaler) Refresh(ctx context.Context, in *autoscaler.RefreshRequest, opts ...grpc.CallOption) (*autoscaler.RefreshResponse, error) {
	return nil, nil
}

// NodeGroupTargetSize returns the current target size of the node group. It is possible
// that the number of nodes in Kubernetes is different at the moment but should be equal
// to the size of a node group once everything stabilizes (new nodes finish startup and
// registration or removed nodes are deleted completely).
func (a *Autoscaler) NodeGroupTargetSize(ctx context.Context, in *autoscaler.NodeGroupTargetSizeRequest, opts ...grpc.CallOption) (*autoscaler.NodeGroupTargetSizeResponse, error) {
	return nil, nil
}

// NodeGroupIncreaseSize increases the size of the node group. To delete a node you need
// to explicitly name it and use NodeGroupDeleteNodes. This function should wait until
// node group size is updated.
func (a *Autoscaler) NodeGroupIncreaseSize(ctx context.Context, in *autoscaler.NodeGroupIncreaseSizeRequest, opts ...grpc.CallOption) (*autoscaler.NodeGroupIncreaseSizeResponse, error) {
	return nil, nil
}

// NodeGroupDeleteNodes deletes nodes from this node group (and also decreasing the size
// of the node group with that). Error is returned either on failure or if the given node
// doesn't belong to this node group. This function should wait until node group size is updated.
func (a *Autoscaler) NodeGroupDeleteNodes(ctx context.Context, in *autoscaler.NodeGroupDeleteNodesRequest, opts ...grpc.CallOption) (*autoscaler.NodeGroupDeleteNodesResponse, error) {
	return nil, nil
}

// NodeGroupDecreaseTargetSize decreases the target size of the node group. This function
// doesn't permit to delete any existing node and can be used only to reduce the request
// for new nodes that have not been yet fulfilled. Delta should be negative. It is assumed
// that cloud provider will not delete the existing nodes if the size when there is an option
// to just decrease the target.
func (a *Autoscaler) NodeGroupDecreaseTargetSize(ctx context.Context, in *autoscaler.NodeGroupDecreaseTargetSizeRequest, opts ...grpc.CallOption) (*autoscaler.NodeGroupDecreaseTargetSizeResponse, error) {
	return nil, nil
}

// NodeGroupNodes returns a list of all nodes that belong to this node group.
func (a *Autoscaler) NodeGroupNodes(ctx context.Context, in *autoscaler.NodeGroupNodesRequest, opts ...grpc.CallOption) (*autoscaler.NodeGroupNodesResponse, error) {
	return nil, nil
}

// NodeGroupTemplateNodeInfo returns a structure of an empty (as if just started) node,
// with all of the labels, capacity and allocatable information. This will be used in
// scale-up simulations to predict what would a new node look like if a node group was expanded.
// Implementation optional: if unimplemented return error code 12 (for `Unimplemented`)
func (a *Autoscaler) NodeGroupTemplateNodeInfo(ctx context.Context, in *autoscaler.NodeGroupTemplateNodeInfoRequest, opts ...grpc.CallOption) (*autoscaler.NodeGroupTemplateNodeInfoResponse, error) {
	return nil, nil
}

// GetOptions returns NodeGroupAutoscalingOptions that should be used for this particular
// NodeGroup.
// Implementation optional: if unimplemented return error code 12 (for `Unimplemented`)
func (a *Autoscaler) NodeGroupGetOptions(ctx context.Context, in *autoscaler.NodeGroupAutoscalingOptionsRequest, opts ...grpc.CallOption) (*autoscaler.NodeGroupAutoscalingOptionsResponse, error) {
	return nil, nil
}

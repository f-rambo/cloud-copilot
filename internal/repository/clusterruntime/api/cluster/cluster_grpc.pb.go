// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.27.1
// source: internal/repository/clusterruntime/api/cluster/cluster.proto

package cluster

import (
	context "context"
	biz "github.com/f-rambo/cloud-copilot/internal/biz"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	ClusterInterface_CurrentCluster_FullMethodName   = "/cluster.ClusterInterface/CurrentCluster"
	ClusterInterface_HandlerNodes_FullMethodName     = "/cluster.ClusterInterface/HandlerNodes"
	ClusterInterface_MigrateToCluster_FullMethodName = "/cluster.ClusterInterface/MigrateToCluster"
)

// ClusterInterfaceClient is the client API for ClusterInterface service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ClusterInterfaceClient interface {
	CurrentCluster(ctx context.Context, in *biz.Cluster, opts ...grpc.CallOption) (*biz.Cluster, error)
	HandlerNodes(ctx context.Context, in *biz.Cluster, opts ...grpc.CallOption) (*biz.Cluster, error)
	MigrateToCluster(ctx context.Context, in *biz.Cluster, opts ...grpc.CallOption) (*biz.Cluster, error)
}

type clusterInterfaceClient struct {
	cc grpc.ClientConnInterface
}

func NewClusterInterfaceClient(cc grpc.ClientConnInterface) ClusterInterfaceClient {
	return &clusterInterfaceClient{cc}
}

func (c *clusterInterfaceClient) CurrentCluster(ctx context.Context, in *biz.Cluster, opts ...grpc.CallOption) (*biz.Cluster, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(biz.Cluster)
	err := c.cc.Invoke(ctx, ClusterInterface_CurrentCluster_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clusterInterfaceClient) HandlerNodes(ctx context.Context, in *biz.Cluster, opts ...grpc.CallOption) (*biz.Cluster, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(biz.Cluster)
	err := c.cc.Invoke(ctx, ClusterInterface_HandlerNodes_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clusterInterfaceClient) MigrateToCluster(ctx context.Context, in *biz.Cluster, opts ...grpc.CallOption) (*biz.Cluster, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(biz.Cluster)
	err := c.cc.Invoke(ctx, ClusterInterface_MigrateToCluster_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ClusterInterfaceServer is the server API for ClusterInterface service.
// All implementations must embed UnimplementedClusterInterfaceServer
// for forward compatibility.
type ClusterInterfaceServer interface {
	CurrentCluster(context.Context, *biz.Cluster) (*biz.Cluster, error)
	HandlerNodes(context.Context, *biz.Cluster) (*biz.Cluster, error)
	MigrateToCluster(context.Context, *biz.Cluster) (*biz.Cluster, error)
	mustEmbedUnimplementedClusterInterfaceServer()
}

// UnimplementedClusterInterfaceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedClusterInterfaceServer struct{}

func (UnimplementedClusterInterfaceServer) CurrentCluster(context.Context, *biz.Cluster) (*biz.Cluster, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CurrentCluster not implemented")
}
func (UnimplementedClusterInterfaceServer) HandlerNodes(context.Context, *biz.Cluster) (*biz.Cluster, error) {
	return nil, status.Errorf(codes.Unimplemented, "method HandlerNodes not implemented")
}
func (UnimplementedClusterInterfaceServer) MigrateToCluster(context.Context, *biz.Cluster) (*biz.Cluster, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MigrateToCluster not implemented")
}
func (UnimplementedClusterInterfaceServer) mustEmbedUnimplementedClusterInterfaceServer() {}
func (UnimplementedClusterInterfaceServer) testEmbeddedByValue()                          {}

// UnsafeClusterInterfaceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ClusterInterfaceServer will
// result in compilation errors.
type UnsafeClusterInterfaceServer interface {
	mustEmbedUnimplementedClusterInterfaceServer()
}

func RegisterClusterInterfaceServer(s grpc.ServiceRegistrar, srv ClusterInterfaceServer) {
	// If the following call pancis, it indicates UnimplementedClusterInterfaceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&ClusterInterface_ServiceDesc, srv)
}

func _ClusterInterface_CurrentCluster_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(biz.Cluster)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClusterInterfaceServer).CurrentCluster(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ClusterInterface_CurrentCluster_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClusterInterfaceServer).CurrentCluster(ctx, req.(*biz.Cluster))
	}
	return interceptor(ctx, in, info, handler)
}

func _ClusterInterface_HandlerNodes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(biz.Cluster)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClusterInterfaceServer).HandlerNodes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ClusterInterface_HandlerNodes_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClusterInterfaceServer).HandlerNodes(ctx, req.(*biz.Cluster))
	}
	return interceptor(ctx, in, info, handler)
}

func _ClusterInterface_MigrateToCluster_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(biz.Cluster)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClusterInterfaceServer).MigrateToCluster(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ClusterInterface_MigrateToCluster_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClusterInterfaceServer).MigrateToCluster(ctx, req.(*biz.Cluster))
	}
	return interceptor(ctx, in, info, handler)
}

// ClusterInterface_ServiceDesc is the grpc.ServiceDesc for ClusterInterface service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ClusterInterface_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "cluster.ClusterInterface",
	HandlerType: (*ClusterInterfaceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CurrentCluster",
			Handler:    _ClusterInterface_CurrentCluster_Handler,
		},
		{
			MethodName: "HandlerNodes",
			Handler:    _ClusterInterface_HandlerNodes_Handler,
		},
		{
			MethodName: "MigrateToCluster",
			Handler:    _ClusterInterface_MigrateToCluster_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "internal/repository/clusterruntime/api/cluster/cluster.proto",
}
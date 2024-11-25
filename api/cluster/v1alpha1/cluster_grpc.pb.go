// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.27.1
// source: api/cluster/v1alpha1/cluster.proto

package v1alpha1

import (
	context "context"
	common "github.com/f-rambo/cloud-copilot/api/common"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	ClusterInterface_Ping_FullMethodName        = "/cluster.v1alpha1.ClusterInterface/Ping"
	ClusterInterface_Get_FullMethodName         = "/cluster.v1alpha1.ClusterInterface/Get"
	ClusterInterface_Save_FullMethodName        = "/cluster.v1alpha1.ClusterInterface/Save"
	ClusterInterface_List_FullMethodName        = "/cluster.v1alpha1.ClusterInterface/List"
	ClusterInterface_Delete_FullMethodName      = "/cluster.v1alpha1.ClusterInterface/Delete"
	ClusterInterface_GetRegions_FullMethodName  = "/cluster.v1alpha1.ClusterInterface/GetRegions"
	ClusterInterface_PollingLogs_FullMethodName = "/cluster.v1alpha1.ClusterInterface/PollingLogs"
	ClusterInterface_GetLogs_FullMethodName     = "/cluster.v1alpha1.ClusterInterface/GetLogs"
)

// ClusterInterfaceClient is the client API for ClusterInterface service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ClusterInterfaceClient interface {
	Ping(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*common.Msg, error)
	Get(ctx context.Context, in *ClusterArgs, opts ...grpc.CallOption) (*Cluster, error)
	Save(ctx context.Context, in *ClusterArgs, opts ...grpc.CallOption) (*Cluster, error)
	List(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*ClusterList, error)
	Delete(ctx context.Context, in *ClusterArgs, opts ...grpc.CallOption) (*common.Msg, error)
	GetRegions(ctx context.Context, in *ClusterArgs, opts ...grpc.CallOption) (*Regions, error)
	PollingLogs(ctx context.Context, in *ClusterLogsRequest, opts ...grpc.CallOption) (*ClusterLogsResponse, error)
	GetLogs(ctx context.Context, opts ...grpc.CallOption) (grpc.BidiStreamingClient[ClusterLogsRequest, ClusterLogsResponse], error)
}

type clusterInterfaceClient struct {
	cc grpc.ClientConnInterface
}

func NewClusterInterfaceClient(cc grpc.ClientConnInterface) ClusterInterfaceClient {
	return &clusterInterfaceClient{cc}
}

func (c *clusterInterfaceClient) Ping(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*common.Msg, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(common.Msg)
	err := c.cc.Invoke(ctx, ClusterInterface_Ping_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clusterInterfaceClient) Get(ctx context.Context, in *ClusterArgs, opts ...grpc.CallOption) (*Cluster, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Cluster)
	err := c.cc.Invoke(ctx, ClusterInterface_Get_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clusterInterfaceClient) Save(ctx context.Context, in *ClusterArgs, opts ...grpc.CallOption) (*Cluster, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Cluster)
	err := c.cc.Invoke(ctx, ClusterInterface_Save_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clusterInterfaceClient) List(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*ClusterList, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ClusterList)
	err := c.cc.Invoke(ctx, ClusterInterface_List_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clusterInterfaceClient) Delete(ctx context.Context, in *ClusterArgs, opts ...grpc.CallOption) (*common.Msg, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(common.Msg)
	err := c.cc.Invoke(ctx, ClusterInterface_Delete_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clusterInterfaceClient) GetRegions(ctx context.Context, in *ClusterArgs, opts ...grpc.CallOption) (*Regions, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Regions)
	err := c.cc.Invoke(ctx, ClusterInterface_GetRegions_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clusterInterfaceClient) PollingLogs(ctx context.Context, in *ClusterLogsRequest, opts ...grpc.CallOption) (*ClusterLogsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ClusterLogsResponse)
	err := c.cc.Invoke(ctx, ClusterInterface_PollingLogs_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clusterInterfaceClient) GetLogs(ctx context.Context, opts ...grpc.CallOption) (grpc.BidiStreamingClient[ClusterLogsRequest, ClusterLogsResponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &ClusterInterface_ServiceDesc.Streams[0], ClusterInterface_GetLogs_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[ClusterLogsRequest, ClusterLogsResponse]{ClientStream: stream}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type ClusterInterface_GetLogsClient = grpc.BidiStreamingClient[ClusterLogsRequest, ClusterLogsResponse]

// ClusterInterfaceServer is the server API for ClusterInterface service.
// All implementations must embed UnimplementedClusterInterfaceServer
// for forward compatibility.
type ClusterInterfaceServer interface {
	Ping(context.Context, *emptypb.Empty) (*common.Msg, error)
	Get(context.Context, *ClusterArgs) (*Cluster, error)
	Save(context.Context, *ClusterArgs) (*Cluster, error)
	List(context.Context, *emptypb.Empty) (*ClusterList, error)
	Delete(context.Context, *ClusterArgs) (*common.Msg, error)
	GetRegions(context.Context, *ClusterArgs) (*Regions, error)
	PollingLogs(context.Context, *ClusterLogsRequest) (*ClusterLogsResponse, error)
	GetLogs(grpc.BidiStreamingServer[ClusterLogsRequest, ClusterLogsResponse]) error
	mustEmbedUnimplementedClusterInterfaceServer()
}

// UnimplementedClusterInterfaceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedClusterInterfaceServer struct{}

func (UnimplementedClusterInterfaceServer) Ping(context.Context, *emptypb.Empty) (*common.Msg, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Ping not implemented")
}
func (UnimplementedClusterInterfaceServer) Get(context.Context, *ClusterArgs) (*Cluster, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Get not implemented")
}
func (UnimplementedClusterInterfaceServer) Save(context.Context, *ClusterArgs) (*Cluster, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Save not implemented")
}
func (UnimplementedClusterInterfaceServer) List(context.Context, *emptypb.Empty) (*ClusterList, error) {
	return nil, status.Errorf(codes.Unimplemented, "method List not implemented")
}
func (UnimplementedClusterInterfaceServer) Delete(context.Context, *ClusterArgs) (*common.Msg, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Delete not implemented")
}
func (UnimplementedClusterInterfaceServer) GetRegions(context.Context, *ClusterArgs) (*Regions, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetRegions not implemented")
}
func (UnimplementedClusterInterfaceServer) PollingLogs(context.Context, *ClusterLogsRequest) (*ClusterLogsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PollingLogs not implemented")
}
func (UnimplementedClusterInterfaceServer) GetLogs(grpc.BidiStreamingServer[ClusterLogsRequest, ClusterLogsResponse]) error {
	return status.Errorf(codes.Unimplemented, "method GetLogs not implemented")
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

func _ClusterInterface_Ping_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClusterInterfaceServer).Ping(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ClusterInterface_Ping_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClusterInterfaceServer).Ping(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _ClusterInterface_Get_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ClusterArgs)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClusterInterfaceServer).Get(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ClusterInterface_Get_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClusterInterfaceServer).Get(ctx, req.(*ClusterArgs))
	}
	return interceptor(ctx, in, info, handler)
}

func _ClusterInterface_Save_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ClusterArgs)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClusterInterfaceServer).Save(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ClusterInterface_Save_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClusterInterfaceServer).Save(ctx, req.(*ClusterArgs))
	}
	return interceptor(ctx, in, info, handler)
}

func _ClusterInterface_List_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClusterInterfaceServer).List(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ClusterInterface_List_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClusterInterfaceServer).List(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _ClusterInterface_Delete_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ClusterArgs)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClusterInterfaceServer).Delete(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ClusterInterface_Delete_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClusterInterfaceServer).Delete(ctx, req.(*ClusterArgs))
	}
	return interceptor(ctx, in, info, handler)
}

func _ClusterInterface_GetRegions_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ClusterArgs)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClusterInterfaceServer).GetRegions(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ClusterInterface_GetRegions_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClusterInterfaceServer).GetRegions(ctx, req.(*ClusterArgs))
	}
	return interceptor(ctx, in, info, handler)
}

func _ClusterInterface_PollingLogs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ClusterLogsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ClusterInterfaceServer).PollingLogs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ClusterInterface_PollingLogs_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ClusterInterfaceServer).PollingLogs(ctx, req.(*ClusterLogsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ClusterInterface_GetLogs_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(ClusterInterfaceServer).GetLogs(&grpc.GenericServerStream[ClusterLogsRequest, ClusterLogsResponse]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type ClusterInterface_GetLogsServer = grpc.BidiStreamingServer[ClusterLogsRequest, ClusterLogsResponse]

// ClusterInterface_ServiceDesc is the grpc.ServiceDesc for ClusterInterface service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ClusterInterface_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "cluster.v1alpha1.ClusterInterface",
	HandlerType: (*ClusterInterfaceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Ping",
			Handler:    _ClusterInterface_Ping_Handler,
		},
		{
			MethodName: "Get",
			Handler:    _ClusterInterface_Get_Handler,
		},
		{
			MethodName: "Save",
			Handler:    _ClusterInterface_Save_Handler,
		},
		{
			MethodName: "List",
			Handler:    _ClusterInterface_List_Handler,
		},
		{
			MethodName: "Delete",
			Handler:    _ClusterInterface_Delete_Handler,
		},
		{
			MethodName: "GetRegions",
			Handler:    _ClusterInterface_GetRegions_Handler,
		},
		{
			MethodName: "PollingLogs",
			Handler:    _ClusterInterface_PollingLogs_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "GetLogs",
			Handler:       _ClusterInterface_GetLogs_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "api/cluster/v1alpha1/cluster.proto",
}

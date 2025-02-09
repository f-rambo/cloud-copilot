// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.27.1
// source: internal/repository/clusterruntime/api/service/service.proto

package service

import (
	context "context"
	common "github.com/f-rambo/cloud-copilot/api/common"
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
	ServiceInterface_Create_FullMethodName             = "/clusterruntime.api.service.ServiceInterface/Create"
	ServiceInterface_GenerateCIWorkflow_FullMethodName = "/clusterruntime.api.service.ServiceInterface/GenerateCIWorkflow"
)

// ServiceInterfaceClient is the client API for ServiceInterface service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ServiceInterfaceClient interface {
	Create(ctx context.Context, in *CreateReq, opts ...grpc.CallOption) (*common.Msg, error)
	GenerateCIWorkflow(ctx context.Context, in *biz.Service, opts ...grpc.CallOption) (*GenerateCIWorkflowResponse, error)
}

type serviceInterfaceClient struct {
	cc grpc.ClientConnInterface
}

func NewServiceInterfaceClient(cc grpc.ClientConnInterface) ServiceInterfaceClient {
	return &serviceInterfaceClient{cc}
}

func (c *serviceInterfaceClient) Create(ctx context.Context, in *CreateReq, opts ...grpc.CallOption) (*common.Msg, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(common.Msg)
	err := c.cc.Invoke(ctx, ServiceInterface_Create_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serviceInterfaceClient) GenerateCIWorkflow(ctx context.Context, in *biz.Service, opts ...grpc.CallOption) (*GenerateCIWorkflowResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GenerateCIWorkflowResponse)
	err := c.cc.Invoke(ctx, ServiceInterface_GenerateCIWorkflow_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ServiceInterfaceServer is the server API for ServiceInterface service.
// All implementations must embed UnimplementedServiceInterfaceServer
// for forward compatibility.
type ServiceInterfaceServer interface {
	Create(context.Context, *CreateReq) (*common.Msg, error)
	GenerateCIWorkflow(context.Context, *biz.Service) (*GenerateCIWorkflowResponse, error)
	mustEmbedUnimplementedServiceInterfaceServer()
}

// UnimplementedServiceInterfaceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedServiceInterfaceServer struct{}

func (UnimplementedServiceInterfaceServer) Create(context.Context, *CreateReq) (*common.Msg, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Create not implemented")
}
func (UnimplementedServiceInterfaceServer) GenerateCIWorkflow(context.Context, *biz.Service) (*GenerateCIWorkflowResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GenerateCIWorkflow not implemented")
}
func (UnimplementedServiceInterfaceServer) mustEmbedUnimplementedServiceInterfaceServer() {}
func (UnimplementedServiceInterfaceServer) testEmbeddedByValue()                          {}

// UnsafeServiceInterfaceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ServiceInterfaceServer will
// result in compilation errors.
type UnsafeServiceInterfaceServer interface {
	mustEmbedUnimplementedServiceInterfaceServer()
}

func RegisterServiceInterfaceServer(s grpc.ServiceRegistrar, srv ServiceInterfaceServer) {
	// If the following call pancis, it indicates UnimplementedServiceInterfaceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&ServiceInterface_ServiceDesc, srv)
}

func _ServiceInterface_Create_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CreateReq)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceInterfaceServer).Create(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ServiceInterface_Create_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceInterfaceServer).Create(ctx, req.(*CreateReq))
	}
	return interceptor(ctx, in, info, handler)
}

func _ServiceInterface_GenerateCIWorkflow_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(biz.Service)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServiceInterfaceServer).GenerateCIWorkflow(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: ServiceInterface_GenerateCIWorkflow_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServiceInterfaceServer).GenerateCIWorkflow(ctx, req.(*biz.Service))
	}
	return interceptor(ctx, in, info, handler)
}

// ServiceInterface_ServiceDesc is the grpc.ServiceDesc for ServiceInterface service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ServiceInterface_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "clusterruntime.api.service.ServiceInterface",
	HandlerType: (*ServiceInterfaceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Create",
			Handler:    _ServiceInterface_Create_Handler,
		},
		{
			MethodName: "GenerateCIWorkflow",
			Handler:    _ServiceInterface_GenerateCIWorkflow_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "internal/repository/clusterruntime/api/service/service.proto",
}

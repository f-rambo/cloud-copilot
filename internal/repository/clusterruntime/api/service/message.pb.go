// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        v5.27.1
// source: internal/repository/clusterruntime/api/service/message.proto

package service

import (
	biz "github.com/f-rambo/cloud-copilot/internal/biz"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type GenerateCIWorkflowResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	CiWorkflow *biz.Workflow `protobuf:"bytes,1,opt,name=ci_workflow,json=ciWorkflow,proto3" json:"ci_workflow,omitempty"`
	CdWorkflow *biz.Workflow `protobuf:"bytes,2,opt,name=cd_workflow,json=cdWorkflow,proto3" json:"cd_workflow,omitempty"`
}

func (x *GenerateCIWorkflowResponse) Reset() {
	*x = GenerateCIWorkflowResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_repository_clusterruntime_api_service_message_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GenerateCIWorkflowResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GenerateCIWorkflowResponse) ProtoMessage() {}

func (x *GenerateCIWorkflowResponse) ProtoReflect() protoreflect.Message {
	mi := &file_internal_repository_clusterruntime_api_service_message_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GenerateCIWorkflowResponse.ProtoReflect.Descriptor instead.
func (*GenerateCIWorkflowResponse) Descriptor() ([]byte, []int) {
	return file_internal_repository_clusterruntime_api_service_message_proto_rawDescGZIP(), []int{0}
}

func (x *GenerateCIWorkflowResponse) GetCiWorkflow() *biz.Workflow {
	if x != nil {
		return x.CiWorkflow
	}
	return nil
}

func (x *GenerateCIWorkflowResponse) GetCdWorkflow() *biz.Workflow {
	if x != nil {
		return x.CdWorkflow
	}
	return nil
}

type CreateReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Namespace string        `protobuf:"bytes,1,opt,name=namespace,proto3" json:"namespace,omitempty"`
	Workflow  *biz.Workflow `protobuf:"bytes,2,opt,name=workflow,proto3" json:"workflow,omitempty"`
}

func (x *CreateReq) Reset() {
	*x = CreateReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_repository_clusterruntime_api_service_message_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CreateReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateReq) ProtoMessage() {}

func (x *CreateReq) ProtoReflect() protoreflect.Message {
	mi := &file_internal_repository_clusterruntime_api_service_message_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use CreateReq.ProtoReflect.Descriptor instead.
func (*CreateReq) Descriptor() ([]byte, []int) {
	return file_internal_repository_clusterruntime_api_service_message_proto_rawDescGZIP(), []int{1}
}

func (x *CreateReq) GetNamespace() string {
	if x != nil {
		return x.Namespace
	}
	return ""
}

func (x *CreateReq) GetWorkflow() *biz.Workflow {
	if x != nil {
		return x.Workflow
	}
	return nil
}

var File_internal_repository_clusterruntime_api_service_message_proto protoreflect.FileDescriptor

var file_internal_repository_clusterruntime_api_service_message_proto_rawDesc = []byte{
	0x0a, 0x3c, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x72, 0x65, 0x70, 0x6f, 0x73,
	0x69, 0x74, 0x6f, 0x72, 0x79, 0x2f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x72, 0x75, 0x6e,
	0x74, 0x69, 0x6d, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65,
	0x2f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x07,
	0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x1a, 0x1a, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61,
	0x6c, 0x2f, 0x62, 0x69, 0x7a, 0x2f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x22, 0x8c, 0x01, 0x0a, 0x1a, 0x47, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74, 0x65,
	0x43, 0x49, 0x57, 0x6f, 0x72, 0x6b, 0x66, 0x6c, 0x6f, 0x77, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e,
	0x73, 0x65, 0x12, 0x36, 0x0a, 0x0b, 0x63, 0x69, 0x5f, 0x77, 0x6f, 0x72, 0x6b, 0x66, 0x6c, 0x6f,
	0x77, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x15, 0x2e, 0x62, 0x69, 0x7a, 0x2e, 0x73, 0x65,
	0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x57, 0x6f, 0x72, 0x6b, 0x66, 0x6c, 0x6f, 0x77, 0x52, 0x0a,
	0x63, 0x69, 0x57, 0x6f, 0x72, 0x6b, 0x66, 0x6c, 0x6f, 0x77, 0x12, 0x36, 0x0a, 0x0b, 0x63, 0x64,
	0x5f, 0x77, 0x6f, 0x72, 0x6b, 0x66, 0x6c, 0x6f, 0x77, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x15, 0x2e, 0x62, 0x69, 0x7a, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x57, 0x6f,
	0x72, 0x6b, 0x66, 0x6c, 0x6f, 0x77, 0x52, 0x0a, 0x63, 0x64, 0x57, 0x6f, 0x72, 0x6b, 0x66, 0x6c,
	0x6f, 0x77, 0x22, 0x5c, 0x0a, 0x09, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x52, 0x65, 0x71, 0x12,
	0x1c, 0x0a, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x70, 0x61, 0x63, 0x65, 0x12, 0x31, 0x0a,
	0x08, 0x77, 0x6f, 0x72, 0x6b, 0x66, 0x6c, 0x6f, 0x77, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x15, 0x2e, 0x62, 0x69, 0x7a, 0x2e, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x57, 0x6f,
	0x72, 0x6b, 0x66, 0x6c, 0x6f, 0x77, 0x52, 0x08, 0x77, 0x6f, 0x72, 0x6b, 0x66, 0x6c, 0x6f, 0x77,
	0x42, 0x52, 0x5a, 0x50, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x66,
	0x2d, 0x72, 0x61, 0x6d, 0x62, 0x6f, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x2d, 0x63, 0x6f, 0x70,
	0x69, 0x6c, 0x6f, 0x74, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x72, 0x65,
	0x70, 0x6f, 0x73, 0x69, 0x74, 0x6f, 0x72, 0x79, 0x2f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72,
	0x72, 0x75, 0x6e, 0x74, 0x69, 0x6d, 0x65, 0x2f, 0x61, 0x70, 0x69, 0x2f, 0x73, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x3b, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_internal_repository_clusterruntime_api_service_message_proto_rawDescOnce sync.Once
	file_internal_repository_clusterruntime_api_service_message_proto_rawDescData = file_internal_repository_clusterruntime_api_service_message_proto_rawDesc
)

func file_internal_repository_clusterruntime_api_service_message_proto_rawDescGZIP() []byte {
	file_internal_repository_clusterruntime_api_service_message_proto_rawDescOnce.Do(func() {
		file_internal_repository_clusterruntime_api_service_message_proto_rawDescData = protoimpl.X.CompressGZIP(file_internal_repository_clusterruntime_api_service_message_proto_rawDescData)
	})
	return file_internal_repository_clusterruntime_api_service_message_proto_rawDescData
}

var file_internal_repository_clusterruntime_api_service_message_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_internal_repository_clusterruntime_api_service_message_proto_goTypes = []any{
	(*GenerateCIWorkflowResponse)(nil), // 0: service.GenerateCIWorkflowResponse
	(*CreateReq)(nil),                  // 1: service.CreateReq
	(*biz.Workflow)(nil),               // 2: biz.service.Workflow
}
var file_internal_repository_clusterruntime_api_service_message_proto_depIdxs = []int32{
	2, // 0: service.GenerateCIWorkflowResponse.ci_workflow:type_name -> biz.service.Workflow
	2, // 1: service.GenerateCIWorkflowResponse.cd_workflow:type_name -> biz.service.Workflow
	2, // 2: service.CreateReq.workflow:type_name -> biz.service.Workflow
	3, // [3:3] is the sub-list for method output_type
	3, // [3:3] is the sub-list for method input_type
	3, // [3:3] is the sub-list for extension type_name
	3, // [3:3] is the sub-list for extension extendee
	0, // [0:3] is the sub-list for field type_name
}

func init() { file_internal_repository_clusterruntime_api_service_message_proto_init() }
func file_internal_repository_clusterruntime_api_service_message_proto_init() {
	if File_internal_repository_clusterruntime_api_service_message_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_internal_repository_clusterruntime_api_service_message_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*GenerateCIWorkflowResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_repository_clusterruntime_api_service_message_proto_msgTypes[1].Exporter = func(v any, i int) any {
			switch v := v.(*CreateReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_internal_repository_clusterruntime_api_service_message_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_internal_repository_clusterruntime_api_service_message_proto_goTypes,
		DependencyIndexes: file_internal_repository_clusterruntime_api_service_message_proto_depIdxs,
		MessageInfos:      file_internal_repository_clusterruntime_api_service_message_proto_msgTypes,
	}.Build()
	File_internal_repository_clusterruntime_api_service_message_proto = out.File
	file_internal_repository_clusterruntime_api_service_message_proto_rawDesc = nil
	file_internal_repository_clusterruntime_api_service_message_proto_goTypes = nil
	file_internal_repository_clusterruntime_api_service_message_proto_depIdxs = nil
}
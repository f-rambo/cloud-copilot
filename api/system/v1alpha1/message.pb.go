// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.34.2
// 	protoc        v5.27.1
// source: api/system/v1alpha1/message.proto

package v1alpha1

import (
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

type System struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id         int64   `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Os         string  `protobuf:"bytes,2,opt,name=os,proto3" json:"os,omitempty"`
	Arch       string  `protobuf:"bytes,3,opt,name=arch,proto3" json:"arch,omitempty"`
	Cpu        int32   `protobuf:"varint,4,opt,name=cpu,proto3" json:"cpu,omitempty"`
	Memory     float64 `protobuf:"fixed64,5,opt,name=memory,proto3" json:"memory,omitempty"`
	Gpu        int32   `protobuf:"varint,6,opt,name=gpu,proto3" json:"gpu,omitempty"`
	GpuSpec    string  `protobuf:"bytes,7,opt,name=gpu_spec,json=gpuSpec,proto3" json:"gpu_spec,omitempty"`
	DataDisk   int32   `protobuf:"varint,8,opt,name=data_disk,json=dataDisk,proto3" json:"data_disk,omitempty"`
	Kernel     string  `protobuf:"bytes,9,opt,name=kernel,proto3" json:"kernel,omitempty"`
	Container  string  `protobuf:"bytes,10,opt,name=container,proto3" json:"container,omitempty"`
	Kubelet    string  `protobuf:"bytes,11,opt,name=kubelet,proto3" json:"kubelet,omitempty"`
	KubeProxy  string  `protobuf:"bytes,12,opt,name=kube_proxy,json=kubeProxy,proto3" json:"kube_proxy,omitempty"`
	InternalIp string  `protobuf:"bytes,13,opt,name=internal_ip,json=internalIp,proto3" json:"internal_ip,omitempty"`
}

func (x *System) Reset() {
	*x = System{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_system_v1alpha1_message_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *System) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*System) ProtoMessage() {}

func (x *System) ProtoReflect() protoreflect.Message {
	mi := &file_api_system_v1alpha1_message_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use System.ProtoReflect.Descriptor instead.
func (*System) Descriptor() ([]byte, []int) {
	return file_api_system_v1alpha1_message_proto_rawDescGZIP(), []int{0}
}

func (x *System) GetId() int64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *System) GetOs() string {
	if x != nil {
		return x.Os
	}
	return ""
}

func (x *System) GetArch() string {
	if x != nil {
		return x.Arch
	}
	return ""
}

func (x *System) GetCpu() int32 {
	if x != nil {
		return x.Cpu
	}
	return 0
}

func (x *System) GetMemory() float64 {
	if x != nil {
		return x.Memory
	}
	return 0
}

func (x *System) GetGpu() int32 {
	if x != nil {
		return x.Gpu
	}
	return 0
}

func (x *System) GetGpuSpec() string {
	if x != nil {
		return x.GpuSpec
	}
	return ""
}

func (x *System) GetDataDisk() int32 {
	if x != nil {
		return x.DataDisk
	}
	return 0
}

func (x *System) GetKernel() string {
	if x != nil {
		return x.Kernel
	}
	return ""
}

func (x *System) GetContainer() string {
	if x != nil {
		return x.Container
	}
	return ""
}

func (x *System) GetKubelet() string {
	if x != nil {
		return x.Kubelet
	}
	return ""
}

func (x *System) GetKubeProxy() string {
	if x != nil {
		return x.KubeProxy
	}
	return ""
}

func (x *System) GetInternalIp() string {
	if x != nil {
		return x.InternalIp
	}
	return ""
}

var File_api_system_v1alpha1_message_proto protoreflect.FileDescriptor

var file_api_system_v1alpha1_message_proto_rawDesc = []byte{
	0x0a, 0x21, 0x61, 0x70, 0x69, 0x2f, 0x73, 0x79, 0x73, 0x74, 0x65, 0x6d, 0x2f, 0x76, 0x31, 0x61,
	0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x0f, 0x73, 0x79, 0x73, 0x74, 0x65, 0x6d, 0x2e, 0x76, 0x31, 0x61, 0x6c,
	0x70, 0x68, 0x61, 0x31, 0x22, 0xc0, 0x02, 0x0a, 0x06, 0x53, 0x79, 0x73, 0x74, 0x65, 0x6d, 0x12,
	0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52, 0x02, 0x69, 0x64, 0x12,
	0x0e, 0x0a, 0x02, 0x6f, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02, 0x6f, 0x73, 0x12,
	0x12, 0x0a, 0x04, 0x61, 0x72, 0x63, 0x68, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x61,
	0x72, 0x63, 0x68, 0x12, 0x10, 0x0a, 0x03, 0x63, 0x70, 0x75, 0x18, 0x04, 0x20, 0x01, 0x28, 0x05,
	0x52, 0x03, 0x63, 0x70, 0x75, 0x12, 0x16, 0x0a, 0x06, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79, 0x18,
	0x05, 0x20, 0x01, 0x28, 0x01, 0x52, 0x06, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79, 0x12, 0x10, 0x0a,
	0x03, 0x67, 0x70, 0x75, 0x18, 0x06, 0x20, 0x01, 0x28, 0x05, 0x52, 0x03, 0x67, 0x70, 0x75, 0x12,
	0x19, 0x0a, 0x08, 0x67, 0x70, 0x75, 0x5f, 0x73, 0x70, 0x65, 0x63, 0x18, 0x07, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x07, 0x67, 0x70, 0x75, 0x53, 0x70, 0x65, 0x63, 0x12, 0x1b, 0x0a, 0x09, 0x64, 0x61,
	0x74, 0x61, 0x5f, 0x64, 0x69, 0x73, 0x6b, 0x18, 0x08, 0x20, 0x01, 0x28, 0x05, 0x52, 0x08, 0x64,
	0x61, 0x74, 0x61, 0x44, 0x69, 0x73, 0x6b, 0x12, 0x16, 0x0a, 0x06, 0x6b, 0x65, 0x72, 0x6e, 0x65,
	0x6c, 0x18, 0x09, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x6b, 0x65, 0x72, 0x6e, 0x65, 0x6c, 0x12,
	0x1c, 0x0a, 0x09, 0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x18, 0x0a, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x09, 0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65, 0x72, 0x12, 0x18, 0x0a,
	0x07, 0x6b, 0x75, 0x62, 0x65, 0x6c, 0x65, 0x74, 0x18, 0x0b, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07,
	0x6b, 0x75, 0x62, 0x65, 0x6c, 0x65, 0x74, 0x12, 0x1d, 0x0a, 0x0a, 0x6b, 0x75, 0x62, 0x65, 0x5f,
	0x70, 0x72, 0x6f, 0x78, 0x79, 0x18, 0x0c, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x6b, 0x75, 0x62,
	0x65, 0x50, 0x72, 0x6f, 0x78, 0x79, 0x12, 0x1f, 0x0a, 0x0b, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e,
	0x61, 0x6c, 0x5f, 0x69, 0x70, 0x18, 0x0d, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x69, 0x6e, 0x74,
	0x65, 0x72, 0x6e, 0x61, 0x6c, 0x49, 0x70, 0x42, 0x1e, 0x5a, 0x1c, 0x61, 0x70, 0x69, 0x2f, 0x73,
	0x79, 0x73, 0x74, 0x65, 0x6d, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x3b, 0x76,
	0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_api_system_v1alpha1_message_proto_rawDescOnce sync.Once
	file_api_system_v1alpha1_message_proto_rawDescData = file_api_system_v1alpha1_message_proto_rawDesc
)

func file_api_system_v1alpha1_message_proto_rawDescGZIP() []byte {
	file_api_system_v1alpha1_message_proto_rawDescOnce.Do(func() {
		file_api_system_v1alpha1_message_proto_rawDescData = protoimpl.X.CompressGZIP(file_api_system_v1alpha1_message_proto_rawDescData)
	})
	return file_api_system_v1alpha1_message_proto_rawDescData
}

var file_api_system_v1alpha1_message_proto_msgTypes = make([]protoimpl.MessageInfo, 1)
var file_api_system_v1alpha1_message_proto_goTypes = []any{
	(*System)(nil), // 0: system.v1alpha1.System
}
var file_api_system_v1alpha1_message_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_api_system_v1alpha1_message_proto_init() }
func file_api_system_v1alpha1_message_proto_init() {
	if File_api_system_v1alpha1_message_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_api_system_v1alpha1_message_proto_msgTypes[0].Exporter = func(v any, i int) any {
			switch v := v.(*System); i {
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
			RawDescriptor: file_api_system_v1alpha1_message_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   1,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_api_system_v1alpha1_message_proto_goTypes,
		DependencyIndexes: file_api_system_v1alpha1_message_proto_depIdxs,
		MessageInfos:      file_api_system_v1alpha1_message_proto_msgTypes,
	}.Build()
	File_api_system_v1alpha1_message_proto = out.File
	file_api_system_v1alpha1_message_proto_rawDesc = nil
	file_api_system_v1alpha1_message_proto_goTypes = nil
	file_api_system_v1alpha1_message_proto_depIdxs = nil
}
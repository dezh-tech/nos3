// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.33.0
// 	protoc        v3.12.4
// source: proto/log.proto

package gen

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

type AddLogRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Message string `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
	Stack   string `protobuf:"bytes,2,opt,name=stack,proto3" json:"stack,omitempty"`
}

func (x *AddLogRequest) Reset() {
	*x = AddLogRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_log_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AddLogRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddLogRequest) ProtoMessage() {}

func (x *AddLogRequest) ProtoReflect() protoreflect.Message {
	mi := &file_proto_log_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddLogRequest.ProtoReflect.Descriptor instead.
func (*AddLogRequest) Descriptor() ([]byte, []int) {
	return file_proto_log_proto_rawDescGZIP(), []int{0}
}

func (x *AddLogRequest) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

func (x *AddLogRequest) GetStack() string {
	if x != nil {
		return x.Stack
	}
	return ""
}

type AddLogResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Success bool    `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	Message *string `protobuf:"bytes,2,opt,name=message,proto3,oneof" json:"message,omitempty"`
}

func (x *AddLogResponse) Reset() {
	*x = AddLogResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_proto_log_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *AddLogResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*AddLogResponse) ProtoMessage() {}

func (x *AddLogResponse) ProtoReflect() protoreflect.Message {
	mi := &file_proto_log_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use AddLogResponse.ProtoReflect.Descriptor instead.
func (*AddLogResponse) Descriptor() ([]byte, []int) {
	return file_proto_log_proto_rawDescGZIP(), []int{1}
}

func (x *AddLogResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *AddLogResponse) GetMessage() string {
	if x != nil && x.Message != nil {
		return *x.Message
	}
	return ""
}

var File_proto_log_proto protoreflect.FileDescriptor

var file_proto_log_proto_rawDesc = []byte{
	0x0a, 0x0f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x6c, 0x6f, 0x67, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x12, 0x0a, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x22, 0x3f, 0x0a,
	0x0d, 0x41, 0x64, 0x64, 0x4c, 0x6f, 0x67, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x18,
	0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x73, 0x74, 0x61, 0x63,
	0x6b, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x73, 0x74, 0x61, 0x63, 0x6b, 0x22, 0x55,
	0x0a, 0x0e, 0x41, 0x64, 0x64, 0x4c, 0x6f, 0x67, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x18, 0x0a, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x08, 0x52, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x12, 0x1d, 0x0a, 0x07, 0x6d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x07, 0x6d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x88, 0x01, 0x01, 0x42, 0x0a, 0x0a, 0x08, 0x5f, 0x6d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x32, 0x46, 0x0a, 0x03, 0x4c, 0x6f, 0x67, 0x12, 0x3f, 0x0a, 0x06,
	0x41, 0x64, 0x64, 0x4c, 0x6f, 0x67, 0x12, 0x19, 0x2e, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x72,
	0x2e, 0x76, 0x31, 0x2e, 0x41, 0x64, 0x64, 0x4c, 0x6f, 0x67, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73,
	0x74, 0x1a, 0x1a, 0x2e, 0x6d, 0x61, 0x6e, 0x61, 0x67, 0x65, 0x72, 0x2e, 0x76, 0x31, 0x2e, 0x41,
	0x64, 0x64, 0x4c, 0x6f, 0x67, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x42, 0x43, 0x5a,
	0x41, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x65, 0x7a, 0x68,
	0x2d, 0x74, 0x65, 0x63, 0x68, 0x2f, 0x6e, 0x6f, 0x73, 0x33, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72,
	0x6e, 0x61, 0x6c, 0x2f, 0x69, 0x6e, 0x66, 0x72, 0x61, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x75,
	0x72, 0x65, 0x2f, 0x67, 0x72, 0x70, 0x63, 0x5f, 0x63, 0x6c, 0x69, 0x65, 0x6e, 0x74, 0x2f, 0x67,
	0x65, 0x6e, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_proto_log_proto_rawDescOnce sync.Once
	file_proto_log_proto_rawDescData = file_proto_log_proto_rawDesc
)

func file_proto_log_proto_rawDescGZIP() []byte {
	file_proto_log_proto_rawDescOnce.Do(func() {
		file_proto_log_proto_rawDescData = protoimpl.X.CompressGZIP(file_proto_log_proto_rawDescData)
	})
	return file_proto_log_proto_rawDescData
}

var file_proto_log_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_proto_log_proto_goTypes = []interface{}{
	(*AddLogRequest)(nil),  // 0: manager.v1.AddLogRequest
	(*AddLogResponse)(nil), // 1: manager.v1.AddLogResponse
}
var file_proto_log_proto_depIdxs = []int32{
	0, // 0: manager.v1.Log.AddLog:input_type -> manager.v1.AddLogRequest
	1, // 1: manager.v1.Log.AddLog:output_type -> manager.v1.AddLogResponse
	1, // [1:2] is the sub-list for method output_type
	0, // [0:1] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_proto_log_proto_init() }
func file_proto_log_proto_init() {
	if File_proto_log_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_proto_log_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AddLogRequest); i {
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
		file_proto_log_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*AddLogResponse); i {
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
	file_proto_log_proto_msgTypes[1].OneofWrappers = []interface{}{}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_proto_log_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_proto_log_proto_goTypes,
		DependencyIndexes: file_proto_log_proto_depIdxs,
		MessageInfos:      file_proto_log_proto_msgTypes,
	}.Build()
	File_proto_log_proto = out.File
	file_proto_log_proto_rawDesc = nil
	file_proto_log_proto_goTypes = nil
	file_proto_log_proto_depIdxs = nil
}

// Code generated by protoc-gen-go. DO NOT EDIT.
// source: server.proto

package gitalypb

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type ServerInfoRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ServerInfoRequest) Reset()         { *m = ServerInfoRequest{} }
func (m *ServerInfoRequest) String() string { return proto.CompactTextString(m) }
func (*ServerInfoRequest) ProtoMessage()    {}
func (*ServerInfoRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_ad098daeda4239f7, []int{0}
}

func (m *ServerInfoRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ServerInfoRequest.Unmarshal(m, b)
}
func (m *ServerInfoRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ServerInfoRequest.Marshal(b, m, deterministic)
}
func (m *ServerInfoRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ServerInfoRequest.Merge(m, src)
}
func (m *ServerInfoRequest) XXX_Size() int {
	return xxx_messageInfo_ServerInfoRequest.Size(m)
}
func (m *ServerInfoRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_ServerInfoRequest.DiscardUnknown(m)
}

var xxx_messageInfo_ServerInfoRequest proto.InternalMessageInfo

type ServerInfoResponse struct {
	ServerVersion        string                              `protobuf:"bytes,1,opt,name=server_version,json=serverVersion,proto3" json:"server_version,omitempty"`
	GitVersion           string                              `protobuf:"bytes,2,opt,name=git_version,json=gitVersion,proto3" json:"git_version,omitempty"`
	StorageStatuses      []*ServerInfoResponse_StorageStatus `protobuf:"bytes,3,rep,name=storage_statuses,json=storageStatuses,proto3" json:"storage_statuses,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                            `json:"-"`
	XXX_unrecognized     []byte                              `json:"-"`
	XXX_sizecache        int32                               `json:"-"`
}

func (m *ServerInfoResponse) Reset()         { *m = ServerInfoResponse{} }
func (m *ServerInfoResponse) String() string { return proto.CompactTextString(m) }
func (*ServerInfoResponse) ProtoMessage()    {}
func (*ServerInfoResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_ad098daeda4239f7, []int{1}
}

func (m *ServerInfoResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ServerInfoResponse.Unmarshal(m, b)
}
func (m *ServerInfoResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ServerInfoResponse.Marshal(b, m, deterministic)
}
func (m *ServerInfoResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ServerInfoResponse.Merge(m, src)
}
func (m *ServerInfoResponse) XXX_Size() int {
	return xxx_messageInfo_ServerInfoResponse.Size(m)
}
func (m *ServerInfoResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_ServerInfoResponse.DiscardUnknown(m)
}

var xxx_messageInfo_ServerInfoResponse proto.InternalMessageInfo

func (m *ServerInfoResponse) GetServerVersion() string {
	if m != nil {
		return m.ServerVersion
	}
	return ""
}

func (m *ServerInfoResponse) GetGitVersion() string {
	if m != nil {
		return m.GitVersion
	}
	return ""
}

func (m *ServerInfoResponse) GetStorageStatuses() []*ServerInfoResponse_StorageStatus {
	if m != nil {
		return m.StorageStatuses
	}
	return nil
}

type ServerInfoResponse_StorageStatus struct {
	StorageName          string   `protobuf:"bytes,1,opt,name=storage_name,json=storageName,proto3" json:"storage_name,omitempty"`
	Readable             bool     `protobuf:"varint,2,opt,name=readable,proto3" json:"readable,omitempty"`
	Writeable            bool     `protobuf:"varint,3,opt,name=writeable,proto3" json:"writeable,omitempty"`
	FsType               string   `protobuf:"bytes,4,opt,name=fs_type,json=fsType,proto3" json:"fs_type,omitempty"`
	FilesystemId         string   `protobuf:"bytes,5,opt,name=filesystem_id,json=filesystemId,proto3" json:"filesystem_id,omitempty"`
	ReplicationFactor    uint32   `protobuf:"varint,6,opt,name=replication_factor,json=replicationFactor,proto3" json:"replication_factor,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ServerInfoResponse_StorageStatus) Reset()         { *m = ServerInfoResponse_StorageStatus{} }
func (m *ServerInfoResponse_StorageStatus) String() string { return proto.CompactTextString(m) }
func (*ServerInfoResponse_StorageStatus) ProtoMessage()    {}
func (*ServerInfoResponse_StorageStatus) Descriptor() ([]byte, []int) {
	return fileDescriptor_ad098daeda4239f7, []int{1, 0}
}

func (m *ServerInfoResponse_StorageStatus) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ServerInfoResponse_StorageStatus.Unmarshal(m, b)
}
func (m *ServerInfoResponse_StorageStatus) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ServerInfoResponse_StorageStatus.Marshal(b, m, deterministic)
}
func (m *ServerInfoResponse_StorageStatus) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ServerInfoResponse_StorageStatus.Merge(m, src)
}
func (m *ServerInfoResponse_StorageStatus) XXX_Size() int {
	return xxx_messageInfo_ServerInfoResponse_StorageStatus.Size(m)
}
func (m *ServerInfoResponse_StorageStatus) XXX_DiscardUnknown() {
	xxx_messageInfo_ServerInfoResponse_StorageStatus.DiscardUnknown(m)
}

var xxx_messageInfo_ServerInfoResponse_StorageStatus proto.InternalMessageInfo

func (m *ServerInfoResponse_StorageStatus) GetStorageName() string {
	if m != nil {
		return m.StorageName
	}
	return ""
}

func (m *ServerInfoResponse_StorageStatus) GetReadable() bool {
	if m != nil {
		return m.Readable
	}
	return false
}

func (m *ServerInfoResponse_StorageStatus) GetWriteable() bool {
	if m != nil {
		return m.Writeable
	}
	return false
}

func (m *ServerInfoResponse_StorageStatus) GetFsType() string {
	if m != nil {
		return m.FsType
	}
	return ""
}

func (m *ServerInfoResponse_StorageStatus) GetFilesystemId() string {
	if m != nil {
		return m.FilesystemId
	}
	return ""
}

func (m *ServerInfoResponse_StorageStatus) GetReplicationFactor() uint32 {
	if m != nil {
		return m.ReplicationFactor
	}
	return 0
}

type DiskStatisticsRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DiskStatisticsRequest) Reset()         { *m = DiskStatisticsRequest{} }
func (m *DiskStatisticsRequest) String() string { return proto.CompactTextString(m) }
func (*DiskStatisticsRequest) ProtoMessage()    {}
func (*DiskStatisticsRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_ad098daeda4239f7, []int{2}
}

func (m *DiskStatisticsRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DiskStatisticsRequest.Unmarshal(m, b)
}
func (m *DiskStatisticsRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DiskStatisticsRequest.Marshal(b, m, deterministic)
}
func (m *DiskStatisticsRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DiskStatisticsRequest.Merge(m, src)
}
func (m *DiskStatisticsRequest) XXX_Size() int {
	return xxx_messageInfo_DiskStatisticsRequest.Size(m)
}
func (m *DiskStatisticsRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_DiskStatisticsRequest.DiscardUnknown(m)
}

var xxx_messageInfo_DiskStatisticsRequest proto.InternalMessageInfo

type DiskStatisticsResponse struct {
	StorageStatuses      []*DiskStatisticsResponse_StorageStatus `protobuf:"bytes,1,rep,name=storage_statuses,json=storageStatuses,proto3" json:"storage_statuses,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                                `json:"-"`
	XXX_unrecognized     []byte                                  `json:"-"`
	XXX_sizecache        int32                                   `json:"-"`
}

func (m *DiskStatisticsResponse) Reset()         { *m = DiskStatisticsResponse{} }
func (m *DiskStatisticsResponse) String() string { return proto.CompactTextString(m) }
func (*DiskStatisticsResponse) ProtoMessage()    {}
func (*DiskStatisticsResponse) Descriptor() ([]byte, []int) {
	return fileDescriptor_ad098daeda4239f7, []int{3}
}

func (m *DiskStatisticsResponse) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DiskStatisticsResponse.Unmarshal(m, b)
}
func (m *DiskStatisticsResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DiskStatisticsResponse.Marshal(b, m, deterministic)
}
func (m *DiskStatisticsResponse) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DiskStatisticsResponse.Merge(m, src)
}
func (m *DiskStatisticsResponse) XXX_Size() int {
	return xxx_messageInfo_DiskStatisticsResponse.Size(m)
}
func (m *DiskStatisticsResponse) XXX_DiscardUnknown() {
	xxx_messageInfo_DiskStatisticsResponse.DiscardUnknown(m)
}

var xxx_messageInfo_DiskStatisticsResponse proto.InternalMessageInfo

func (m *DiskStatisticsResponse) GetStorageStatuses() []*DiskStatisticsResponse_StorageStatus {
	if m != nil {
		return m.StorageStatuses
	}
	return nil
}

type DiskStatisticsResponse_StorageStatus struct {
	// When both available and used fields are equal 0 that means that
	// Gitaly was unable to determine storage stats.
	StorageName          string   `protobuf:"bytes,1,opt,name=storage_name,json=storageName,proto3" json:"storage_name,omitempty"`
	Available            int64    `protobuf:"varint,2,opt,name=available,proto3" json:"available,omitempty"`
	Used                 int64    `protobuf:"varint,3,opt,name=used,proto3" json:"used,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DiskStatisticsResponse_StorageStatus) Reset()         { *m = DiskStatisticsResponse_StorageStatus{} }
func (m *DiskStatisticsResponse_StorageStatus) String() string { return proto.CompactTextString(m) }
func (*DiskStatisticsResponse_StorageStatus) ProtoMessage()    {}
func (*DiskStatisticsResponse_StorageStatus) Descriptor() ([]byte, []int) {
	return fileDescriptor_ad098daeda4239f7, []int{3, 0}
}

func (m *DiskStatisticsResponse_StorageStatus) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DiskStatisticsResponse_StorageStatus.Unmarshal(m, b)
}
func (m *DiskStatisticsResponse_StorageStatus) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DiskStatisticsResponse_StorageStatus.Marshal(b, m, deterministic)
}
func (m *DiskStatisticsResponse_StorageStatus) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DiskStatisticsResponse_StorageStatus.Merge(m, src)
}
func (m *DiskStatisticsResponse_StorageStatus) XXX_Size() int {
	return xxx_messageInfo_DiskStatisticsResponse_StorageStatus.Size(m)
}
func (m *DiskStatisticsResponse_StorageStatus) XXX_DiscardUnknown() {
	xxx_messageInfo_DiskStatisticsResponse_StorageStatus.DiscardUnknown(m)
}

var xxx_messageInfo_DiskStatisticsResponse_StorageStatus proto.InternalMessageInfo

func (m *DiskStatisticsResponse_StorageStatus) GetStorageName() string {
	if m != nil {
		return m.StorageName
	}
	return ""
}

func (m *DiskStatisticsResponse_StorageStatus) GetAvailable() int64 {
	if m != nil {
		return m.Available
	}
	return 0
}

func (m *DiskStatisticsResponse_StorageStatus) GetUsed() int64 {
	if m != nil {
		return m.Used
	}
	return 0
}

func init() {
	proto.RegisterType((*ServerInfoRequest)(nil), "gitaly.ServerInfoRequest")
	proto.RegisterType((*ServerInfoResponse)(nil), "gitaly.ServerInfoResponse")
	proto.RegisterType((*ServerInfoResponse_StorageStatus)(nil), "gitaly.ServerInfoResponse.StorageStatus")
	proto.RegisterType((*DiskStatisticsRequest)(nil), "gitaly.DiskStatisticsRequest")
	proto.RegisterType((*DiskStatisticsResponse)(nil), "gitaly.DiskStatisticsResponse")
	proto.RegisterType((*DiskStatisticsResponse_StorageStatus)(nil), "gitaly.DiskStatisticsResponse.StorageStatus")
}

func init() { proto.RegisterFile("server.proto", fileDescriptor_ad098daeda4239f7) }

var fileDescriptor_ad098daeda4239f7 = []byte{
	// 460 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x9c, 0x53, 0xc1, 0x6e, 0xd3, 0x40,
	0x10, 0x95, 0xeb, 0x10, 0x92, 0x49, 0x52, 0xda, 0x41, 0x50, 0x63, 0x15, 0x08, 0x41, 0x48, 0x3e,
	0x50, 0x07, 0x95, 0x3f, 0x40, 0x08, 0xa9, 0x07, 0x38, 0x38, 0x08, 0x04, 0x17, 0x6b, 0x63, 0x8f,
	0xad, 0x15, 0x8e, 0xd7, 0xec, 0x6c, 0x82, 0xf2, 0x11, 0x9c, 0xf9, 0x16, 0xbe, 0x04, 0xf1, 0x2b,
	0x9c, 0x50, 0xd6, 0x49, 0x9c, 0x42, 0x0a, 0x52, 0x2f, 0xd6, 0xce, 0x7b, 0x6f, 0x66, 0x67, 0xdf,
	0x8c, 0xa1, 0xcf, 0xa4, 0x17, 0xa4, 0xc3, 0x4a, 0x2b, 0xa3, 0xb0, 0x9d, 0x4b, 0x23, 0x8a, 0xa5,
	0x0f, 0x85, 0x2c, 0x4d, 0x8d, 0x8d, 0x6e, 0xc3, 0xf1, 0xc4, 0x6a, 0x2e, 0xca, 0x4c, 0x45, 0xf4,
	0x79, 0x4e, 0x6c, 0x46, 0x5f, 0x5d, 0xc0, 0x5d, 0x94, 0x2b, 0x55, 0x32, 0xe1, 0x13, 0x38, 0xac,
	0xeb, 0xc5, 0x0b, 0xd2, 0x2c, 0x55, 0xe9, 0x39, 0x43, 0x27, 0xe8, 0x46, 0x83, 0x1a, 0x7d, 0x57,
	0x83, 0xf8, 0x10, 0x7a, 0xb9, 0x34, 0x5b, 0xcd, 0x81, 0xd5, 0x40, 0x2e, 0xcd, 0x46, 0x30, 0x81,
	0x23, 0x36, 0x4a, 0x8b, 0x9c, 0x62, 0x36, 0xc2, 0xcc, 0x99, 0xd8, 0x73, 0x87, 0x6e, 0xd0, 0x3b,
	0x0f, 0xc2, 0xba, 0xc5, 0xf0, 0xef, 0xdb, 0xc3, 0x49, 0x9d, 0x32, 0xb1, 0x19, 0xd1, 0x2d, 0xde,
	0x0d, 0x89, 0xfd, 0x9f, 0x0e, 0x0c, 0x2e, 0x49, 0xf0, 0x11, 0xf4, 0x37, 0xd7, 0x94, 0x62, 0x46,
	0xeb, 0x66, 0x7b, 0x6b, 0xec, 0x8d, 0x98, 0x11, 0xfa, 0xd0, 0xd1, 0x24, 0x52, 0x31, 0x2d, 0xc8,
	0xf6, 0xd9, 0x89, 0xb6, 0x31, 0x9e, 0x42, 0xf7, 0x8b, 0x96, 0x86, 0x2c, 0xe9, 0x5a, 0xb2, 0x01,
	0xf0, 0x04, 0x6e, 0x66, 0x1c, 0x9b, 0x65, 0x45, 0x5e, 0xcb, 0xd6, 0x6d, 0x67, 0xfc, 0x76, 0x59,
	0x11, 0x3e, 0x86, 0x41, 0x26, 0x0b, 0xe2, 0x25, 0x1b, 0x9a, 0xc5, 0x32, 0xf5, 0x6e, 0x58, 0xba,
	0xdf, 0x80, 0x17, 0x29, 0x9e, 0x01, 0x6a, 0xaa, 0x0a, 0x99, 0x08, 0x23, 0x55, 0x19, 0x67, 0x22,
	0x31, 0x4a, 0x7b, 0xed, 0xa1, 0x13, 0x0c, 0xa2, 0xe3, 0x1d, 0xe6, 0x95, 0x25, 0x46, 0x27, 0x70,
	0xe7, 0xa5, 0xe4, 0x4f, 0xab, 0x77, 0x49, 0x36, 0x32, 0xe1, 0xcd, 0xa0, 0x7e, 0x38, 0x70, 0xf7,
	0x4f, 0x66, 0x3d, 0xac, 0xf7, 0x7b, 0x4c, 0x76, 0xac, 0xc9, 0x4f, 0x37, 0x26, 0xef, 0xcf, 0xfc,
	0x9f, 0xd1, 0xe9, 0x35, 0x7c, 0x3e, 0x85, 0xae, 0x58, 0x08, 0x59, 0x6c, 0x8d, 0x76, 0xa3, 0x06,
	0x40, 0x84, 0xd6, 0x9c, 0x29, 0xb5, 0x26, 0xbb, 0x91, 0x3d, 0x9f, 0x7f, 0x5f, 0x8d, 0xd3, 0x2e,
	0xc1, 0xea, 0x2b, 0x13, 0xc2, 0xd7, 0x00, 0xcd, 0x56, 0xe0, 0xbd, 0x7d, 0x9b, 0x62, 0x4d, 0xf1,
	0xfd, 0xab, 0x97, 0x68, 0xd4, 0xf9, 0xf5, 0x2d, 0x68, 0x75, 0x0e, 0x8e, 0x1c, 0xfc, 0x00, 0x87,
	0x97, 0xdf, 0x8f, 0xf7, 0xaf, 0xf2, 0xa5, 0x2e, 0xfb, 0xe0, 0xdf, 0xb6, 0x35, 0xa5, 0x5f, 0x3c,
	0xfb, 0xb8, 0x92, 0x16, 0x62, 0x1a, 0x26, 0x6a, 0x36, 0xae, 0x8f, 0x67, 0x4a, 0xe7, 0xe3, 0xba,
	0xc0, 0xd8, 0xfe, 0x79, 0xe3, 0x5c, 0xad, 0xe3, 0x6a, 0x3a, 0x6d, 0x5b, 0xe8, 0xf9, 0xef, 0x00,
	0x00, 0x00, 0xff, 0xff, 0x7c, 0x52, 0xb9, 0x17, 0xb0, 0x03, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// ServerServiceClient is the client API for ServerService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type ServerServiceClient interface {
	ServerInfo(ctx context.Context, in *ServerInfoRequest, opts ...grpc.CallOption) (*ServerInfoResponse, error)
	DiskStatistics(ctx context.Context, in *DiskStatisticsRequest, opts ...grpc.CallOption) (*DiskStatisticsResponse, error)
}

type serverServiceClient struct {
	cc *grpc.ClientConn
}

func NewServerServiceClient(cc *grpc.ClientConn) ServerServiceClient {
	return &serverServiceClient{cc}
}

func (c *serverServiceClient) ServerInfo(ctx context.Context, in *ServerInfoRequest, opts ...grpc.CallOption) (*ServerInfoResponse, error) {
	out := new(ServerInfoResponse)
	err := c.cc.Invoke(ctx, "/gitaly.ServerService/ServerInfo", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *serverServiceClient) DiskStatistics(ctx context.Context, in *DiskStatisticsRequest, opts ...grpc.CallOption) (*DiskStatisticsResponse, error) {
	out := new(DiskStatisticsResponse)
	err := c.cc.Invoke(ctx, "/gitaly.ServerService/DiskStatistics", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// ServerServiceServer is the server API for ServerService service.
type ServerServiceServer interface {
	ServerInfo(context.Context, *ServerInfoRequest) (*ServerInfoResponse, error)
	DiskStatistics(context.Context, *DiskStatisticsRequest) (*DiskStatisticsResponse, error)
}

// UnimplementedServerServiceServer can be embedded to have forward compatible implementations.
type UnimplementedServerServiceServer struct {
}

func (*UnimplementedServerServiceServer) ServerInfo(ctx context.Context, req *ServerInfoRequest) (*ServerInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ServerInfo not implemented")
}
func (*UnimplementedServerServiceServer) DiskStatistics(ctx context.Context, req *DiskStatisticsRequest) (*DiskStatisticsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method DiskStatistics not implemented")
}

func RegisterServerServiceServer(s *grpc.Server, srv ServerServiceServer) {
	s.RegisterService(&_ServerService_serviceDesc, srv)
}

func _ServerService_ServerInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ServerInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServerServiceServer).ServerInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gitaly.ServerService/ServerInfo",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServerServiceServer).ServerInfo(ctx, req.(*ServerInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _ServerService_DiskStatistics_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DiskStatisticsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ServerServiceServer).DiskStatistics(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gitaly.ServerService/DiskStatistics",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ServerServiceServer).DiskStatistics(ctx, req.(*DiskStatisticsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _ServerService_serviceDesc = grpc.ServiceDesc{
	ServiceName: "gitaly.ServerService",
	HandlerType: (*ServerServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ServerInfo",
			Handler:    _ServerService_ServerInfo_Handler,
		},
		{
			MethodName: "DiskStatistics",
			Handler:    _ServerService_DiskStatistics_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "server.proto",
}

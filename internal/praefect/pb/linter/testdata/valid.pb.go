// Code generated by protoc-gen-go. DO NOT EDIT.
// source: linter/testdata/valid.proto

package test

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	_ "gitlab.com/gitlab-org/gitaly/internal/praefect/pb"
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

// TestRequest has the required option, so we should not expect it to cause
// a failure
type TestRequest struct {
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *TestRequest) Reset()         { *m = TestRequest{} }
func (m *TestRequest) String() string { return proto.CompactTextString(m) }
func (*TestRequest) ProtoMessage()    {}
func (*TestRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_b81a39d04d27ef36, []int{0}
}

func (m *TestRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TestRequest.Unmarshal(m, b)
}
func (m *TestRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TestRequest.Marshal(b, m, deterministic)
}
func (m *TestRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TestRequest.Merge(m, src)
}
func (m *TestRequest) XXX_Size() int {
	return xxx_messageInfo_TestRequest.Size(m)
}
func (m *TestRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_TestRequest.DiscardUnknown(m)
}

var xxx_messageInfo_TestRequest proto.InternalMessageInfo

func init() {
	proto.RegisterType((*TestRequest)(nil), "test.TestRequest")
}

func init() { proto.RegisterFile("linter/testdata/valid.proto", fileDescriptor_b81a39d04d27ef36) }

var fileDescriptor_b81a39d04d27ef36 = []byte{
	// 95 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0x92, 0xce, 0xc9, 0xcc, 0x2b,
	0x49, 0x2d, 0xd2, 0x2f, 0x49, 0x2d, 0x2e, 0x49, 0x49, 0x2c, 0x49, 0xd4, 0x2f, 0x4b, 0xcc, 0xc9,
	0x4c, 0xd1, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x01, 0x89, 0x4a, 0xf1, 0xe4, 0x17, 0x94,
	0x54, 0x16, 0xa4, 0x42, 0xc4, 0x94, 0x44, 0xb9, 0xb8, 0x43, 0x52, 0x8b, 0x4b, 0x82, 0x52, 0x0b,
	0x4b, 0x53, 0x8b, 0x4b, 0xac, 0xd8, 0x3e, 0x4d, 0xd7, 0x60, 0xe2, 0x60, 0x4a, 0x62, 0x03, 0xcb,
	0x1a, 0x03, 0x02, 0x00, 0x00, 0xff, 0xff, 0x2d, 0x43, 0xaf, 0x4a, 0x50, 0x00, 0x00, 0x00,
}

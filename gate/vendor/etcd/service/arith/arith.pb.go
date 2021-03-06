// Code generated by protoc-gen-go. DO NOT EDIT.
// source: arith.proto

/*
Package arith is a generated protocol buffer package.

It is generated from these files:
	arith.proto

It has these top-level messages:
	Request
	Response
*/
package arith

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

import (
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type Request struct {
	Dig1  int64 `protobuf:"varint,1,opt,name=dig1" json:"dig1,omitempty"`
	Dig2  int64 `protobuf:"varint,2,opt,name=dig2" json:"dig2,omitempty"`
	Count int64 `protobuf:"varint,3,opt,name=count" json:"count,omitempty"`
}

func (m *Request) Reset()                    { *m = Request{} }
func (m *Request) String() string            { return proto.CompactTextString(m) }
func (*Request) ProtoMessage()               {}
func (*Request) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{0} }

func (m *Request) GetDig1() int64 {
	if m != nil {
		return m.Dig1
	}
	return 0
}

func (m *Request) GetDig2() int64 {
	if m != nil {
		return m.Dig2
	}
	return 0
}

func (m *Request) GetCount() int64 {
	if m != nil {
		return m.Count
	}
	return 0
}

type Response struct {
	Result int64 `protobuf:"varint,1,opt,name=result" json:"result,omitempty"`
}

func (m *Response) Reset()                    { *m = Response{} }
func (m *Response) String() string            { return proto.CompactTextString(m) }
func (*Response) ProtoMessage()               {}
func (*Response) Descriptor() ([]byte, []int) { return fileDescriptor0, []int{1} }

func (m *Response) GetResult() int64 {
	if m != nil {
		return m.Result
	}
	return 0
}

func init() {
	proto.RegisterType((*Request)(nil), "arith.Request")
	proto.RegisterType((*Response)(nil), "arith.Response")
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// Client API for Calculator service

type CalculatorClient interface {
	Add(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Response, error)
	Minus(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Response, error)
	Prod(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Response, error)
	Divide(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Response, error)
}

type calculatorClient struct {
	cc *grpc.ClientConn
}

func NewCalculatorClient(cc *grpc.ClientConn) CalculatorClient {
	return &calculatorClient{cc}
}

func (c *calculatorClient) Add(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Response, error) {
	out := new(Response)
	err := grpc.Invoke(ctx, "/arith.Calculator/Add", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *calculatorClient) Minus(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Response, error) {
	out := new(Response)
	err := grpc.Invoke(ctx, "/arith.Calculator/Minus", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *calculatorClient) Prod(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Response, error) {
	out := new(Response)
	err := grpc.Invoke(ctx, "/arith.Calculator/Prod", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *calculatorClient) Divide(ctx context.Context, in *Request, opts ...grpc.CallOption) (*Response, error) {
	out := new(Response)
	err := grpc.Invoke(ctx, "/arith.Calculator/Divide", in, out, c.cc, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Server API for Calculator service

type CalculatorServer interface {
	Add(context.Context, *Request) (*Response, error)
	Minus(context.Context, *Request) (*Response, error)
	Prod(context.Context, *Request) (*Response, error)
	Divide(context.Context, *Request) (*Response, error)
}

func RegisterCalculatorServer(s *grpc.Server, srv CalculatorServer) {
	s.RegisterService(&_Calculator_serviceDesc, srv)
}

func _Calculator_Add_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CalculatorServer).Add(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/arith.Calculator/Add",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CalculatorServer).Add(ctx, req.(*Request))
	}
	return interceptor(ctx, in, info, handler)
}

func _Calculator_Minus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CalculatorServer).Minus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/arith.Calculator/Minus",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CalculatorServer).Minus(ctx, req.(*Request))
	}
	return interceptor(ctx, in, info, handler)
}

func _Calculator_Prod_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CalculatorServer).Prod(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/arith.Calculator/Prod",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CalculatorServer).Prod(ctx, req.(*Request))
	}
	return interceptor(ctx, in, info, handler)
}

func _Calculator_Divide_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Request)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CalculatorServer).Divide(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/arith.Calculator/Divide",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CalculatorServer).Divide(ctx, req.(*Request))
	}
	return interceptor(ctx, in, info, handler)
}

var _Calculator_serviceDesc = grpc.ServiceDesc{
	ServiceName: "arith.Calculator",
	HandlerType: (*CalculatorServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Add",
			Handler:    _Calculator_Add_Handler,
		},
		{
			MethodName: "Minus",
			Handler:    _Calculator_Minus_Handler,
		},
		{
			MethodName: "Prod",
			Handler:    _Calculator_Prod_Handler,
		},
		{
			MethodName: "Divide",
			Handler:    _Calculator_Divide_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "arith.proto",
}

func init() { proto.RegisterFile("arith.proto", fileDescriptor0) }

var fileDescriptor0 = []byte{
	// 191 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xe2, 0x4e, 0x2c, 0xca, 0x2c,
	0xc9, 0xd0, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x17, 0x62, 0x05, 0x73, 0x94, 0xdc, 0xb9, 0xd8, 0x83,
	0x52, 0x0b, 0x4b, 0x53, 0x8b, 0x4b, 0x84, 0x84, 0xb8, 0x58, 0x52, 0x32, 0xd3, 0x0d, 0x25, 0x18,
	0x15, 0x18, 0x35, 0x98, 0x83, 0xc0, 0x6c, 0xa8, 0x98, 0x91, 0x04, 0x13, 0x5c, 0xcc, 0x48, 0x48,
	0x84, 0x8b, 0x35, 0x39, 0xbf, 0x34, 0xaf, 0x44, 0x82, 0x19, 0x2c, 0x08, 0xe1, 0x28, 0x29, 0x71,
	0x71, 0x04, 0xa5, 0x16, 0x17, 0xe4, 0xe7, 0x15, 0xa7, 0x0a, 0x89, 0x71, 0xb1, 0x15, 0xa5, 0x16,
	0x97, 0xe6, 0x94, 0x40, 0xcd, 0x82, 0xf2, 0x8c, 0x76, 0x31, 0x72, 0x71, 0x39, 0x27, 0xe6, 0x24,
	0x97, 0xe6, 0x24, 0x96, 0xe4, 0x17, 0x09, 0x69, 0x70, 0x31, 0x3b, 0xa6, 0xa4, 0x08, 0xf1, 0xe9,
	0x41, 0xdc, 0x05, 0x75, 0x87, 0x14, 0x3f, 0x9c, 0x0f, 0x31, 0x4e, 0x89, 0x41, 0x48, 0x8b, 0x8b,
	0xd5, 0x37, 0x33, 0xaf, 0xb4, 0x98, 0x18, 0xb5, 0x9a, 0x5c, 0x2c, 0x01, 0x45, 0xf9, 0x44, 0x19,
	0xab, 0xcd, 0xc5, 0xe6, 0x92, 0x59, 0x96, 0x99, 0x92, 0x4a, 0x84, 0xe2, 0x24, 0x36, 0x70, 0xb8,
	0x19, 0x03, 0x02, 0x00, 0x00, 0xff, 0xff, 0xef, 0xbc, 0xa1, 0x8b, 0x46, 0x01, 0x00, 0x00,
}

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.19.4
// source: ims.proto

package __

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// InstanceMetaServiceClient is the client API for InstanceMetaService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type InstanceMetaServiceClient interface {
	Status(ctx context.Context, in *Void, opts ...grpc.CallOption) (*StatusReply, error)
	Stop(ctx context.Context, in *Void, opts ...grpc.CallOption) (*StopReply, error)
	AssumeRole(ctx context.Context, in *AssumeRoleRequest, opts ...grpc.CallOption) (*StatusReply, error)
	RetrieveRole(ctx context.Context, in *AssumeRoleRequest, opts ...grpc.CallOption) (*StatusReply, error)
	Config(ctx context.Context, in *Void, opts ...grpc.CallOption) (*ConfigReply, error)
}

type instanceMetaServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewInstanceMetaServiceClient(cc grpc.ClientConnInterface) InstanceMetaServiceClient {
	return &instanceMetaServiceClient{cc}
}

func (c *instanceMetaServiceClient) Status(ctx context.Context, in *Void, opts ...grpc.CallOption) (*StatusReply, error) {
	out := new(StatusReply)
	err := c.cc.Invoke(ctx, "/ims.InstanceMetaService/Status", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *instanceMetaServiceClient) Stop(ctx context.Context, in *Void, opts ...grpc.CallOption) (*StopReply, error) {
	out := new(StopReply)
	err := c.cc.Invoke(ctx, "/ims.InstanceMetaService/Stop", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *instanceMetaServiceClient) AssumeRole(ctx context.Context, in *AssumeRoleRequest, opts ...grpc.CallOption) (*StatusReply, error) {
	out := new(StatusReply)
	err := c.cc.Invoke(ctx, "/ims.InstanceMetaService/AssumeRole", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *instanceMetaServiceClient) RetrieveRole(ctx context.Context, in *AssumeRoleRequest, opts ...grpc.CallOption) (*StatusReply, error) {
	out := new(StatusReply)
	err := c.cc.Invoke(ctx, "/ims.InstanceMetaService/RetrieveRole", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *instanceMetaServiceClient) Config(ctx context.Context, in *Void, opts ...grpc.CallOption) (*ConfigReply, error) {
	out := new(ConfigReply)
	err := c.cc.Invoke(ctx, "/ims.InstanceMetaService/Config", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// InstanceMetaServiceServer is the server API for InstanceMetaService service.
// All implementations must embed UnimplementedInstanceMetaServiceServer
// for forward compatibility
type InstanceMetaServiceServer interface {
	Status(context.Context, *Void) (*StatusReply, error)
	Stop(context.Context, *Void) (*StopReply, error)
	AssumeRole(context.Context, *AssumeRoleRequest) (*StatusReply, error)
	RetrieveRole(context.Context, *AssumeRoleRequest) (*StatusReply, error)
	Config(context.Context, *Void) (*ConfigReply, error)
	mustEmbedUnimplementedInstanceMetaServiceServer()
}

// UnimplementedInstanceMetaServiceServer must be embedded to have forward compatible implementations.
type UnimplementedInstanceMetaServiceServer struct {
}

func (UnimplementedInstanceMetaServiceServer) Status(context.Context, *Void) (*StatusReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Status not implemented")
}
func (UnimplementedInstanceMetaServiceServer) Stop(context.Context, *Void) (*StopReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Stop not implemented")
}
func (UnimplementedInstanceMetaServiceServer) AssumeRole(context.Context, *AssumeRoleRequest) (*StatusReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AssumeRole not implemented")
}
func (UnimplementedInstanceMetaServiceServer) RetrieveRole(context.Context, *AssumeRoleRequest) (*StatusReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RetrieveRole not implemented")
}
func (UnimplementedInstanceMetaServiceServer) Config(context.Context, *Void) (*ConfigReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Config not implemented")
}
func (UnimplementedInstanceMetaServiceServer) mustEmbedUnimplementedInstanceMetaServiceServer() {}

// UnsafeInstanceMetaServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to InstanceMetaServiceServer will
// result in compilation errors.
type UnsafeInstanceMetaServiceServer interface {
	mustEmbedUnimplementedInstanceMetaServiceServer()
}

func RegisterInstanceMetaServiceServer(s grpc.ServiceRegistrar, srv InstanceMetaServiceServer) {
	s.RegisterService(&InstanceMetaService_ServiceDesc, srv)
}

func _InstanceMetaService_Status_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Void)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(InstanceMetaServiceServer).Status(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ims.InstanceMetaService/Status",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(InstanceMetaServiceServer).Status(ctx, req.(*Void))
	}
	return interceptor(ctx, in, info, handler)
}

func _InstanceMetaService_Stop_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Void)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(InstanceMetaServiceServer).Stop(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ims.InstanceMetaService/Stop",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(InstanceMetaServiceServer).Stop(ctx, req.(*Void))
	}
	return interceptor(ctx, in, info, handler)
}

func _InstanceMetaService_AssumeRole_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AssumeRoleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(InstanceMetaServiceServer).AssumeRole(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ims.InstanceMetaService/AssumeRole",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(InstanceMetaServiceServer).AssumeRole(ctx, req.(*AssumeRoleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _InstanceMetaService_RetrieveRole_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AssumeRoleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(InstanceMetaServiceServer).RetrieveRole(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ims.InstanceMetaService/RetrieveRole",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(InstanceMetaServiceServer).RetrieveRole(ctx, req.(*AssumeRoleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _InstanceMetaService_Config_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Void)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(InstanceMetaServiceServer).Config(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/ims.InstanceMetaService/Config",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(InstanceMetaServiceServer).Config(ctx, req.(*Void))
	}
	return interceptor(ctx, in, info, handler)
}

// InstanceMetaService_ServiceDesc is the grpc.ServiceDesc for InstanceMetaService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var InstanceMetaService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "ims.InstanceMetaService",
	HandlerType: (*InstanceMetaServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Status",
			Handler:    _InstanceMetaService_Status_Handler,
		},
		{
			MethodName: "Stop",
			Handler:    _InstanceMetaService_Stop_Handler,
		},
		{
			MethodName: "AssumeRole",
			Handler:    _InstanceMetaService_AssumeRole_Handler,
		},
		{
			MethodName: "RetrieveRole",
			Handler:    _InstanceMetaService_RetrieveRole_Handler,
		},
		{
			MethodName: "Config",
			Handler:    _InstanceMetaService_Config_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "ims.proto",
}
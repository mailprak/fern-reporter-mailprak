// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.27.0
// source: updatetestrun.proto

package updatetestrun

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	TestRunService_UpdateTestRun_FullMethodName = "/updatetestrun.TestRunService/UpdateTestRun"
)

// TestRunServiceClient is the client API for TestRunService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type TestRunServiceClient interface {
	UpdateTestRun(ctx context.Context, in *UpdateTestRunRequest, opts ...grpc.CallOption) (*TestRunResponse, error)
}

type testRunServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewTestRunServiceClient(cc grpc.ClientConnInterface) TestRunServiceClient {
	return &testRunServiceClient{cc}
}

func (c *testRunServiceClient) UpdateTestRun(ctx context.Context, in *UpdateTestRunRequest, opts ...grpc.CallOption) (*TestRunResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(TestRunResponse)
	err := c.cc.Invoke(ctx, TestRunService_UpdateTestRun_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// TestRunServiceServer is the server API for TestRunService service.
// All implementations must embed UnimplementedTestRunServiceServer
// for forward compatibility.
type TestRunServiceServer interface {
	UpdateTestRun(context.Context, *UpdateTestRunRequest) (*TestRunResponse, error)
	mustEmbedUnimplementedTestRunServiceServer()
}

// UnimplementedTestRunServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedTestRunServiceServer struct{}

func (UnimplementedTestRunServiceServer) UpdateTestRun(context.Context, *UpdateTestRunRequest) (*TestRunResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateTestRun not implemented")
}
func (UnimplementedTestRunServiceServer) mustEmbedUnimplementedTestRunServiceServer() {}
func (UnimplementedTestRunServiceServer) testEmbeddedByValue()                        {}

// UnsafeTestRunServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to TestRunServiceServer will
// result in compilation errors.
type UnsafeTestRunServiceServer interface {
	mustEmbedUnimplementedTestRunServiceServer()
}

func RegisterTestRunServiceServer(s grpc.ServiceRegistrar, srv TestRunServiceServer) {
	// If the following call pancis, it indicates UnimplementedTestRunServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&TestRunService_ServiceDesc, srv)
}

func _TestRunService_UpdateTestRun_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(UpdateTestRunRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TestRunServiceServer).UpdateTestRun(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: TestRunService_UpdateTestRun_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TestRunServiceServer).UpdateTestRun(ctx, req.(*UpdateTestRunRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// TestRunService_ServiceDesc is the grpc.ServiceDesc for TestRunService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var TestRunService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "updatetestrun.TestRunService",
	HandlerType: (*TestRunServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "UpdateTestRun",
			Handler:    _TestRunService_UpdateTestRun_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "updatetestrun.proto",
}

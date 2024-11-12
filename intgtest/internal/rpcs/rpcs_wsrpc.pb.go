// Code generated by protoc-gen-go-wsrpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-wsrpc v0.0.1
// - protoc             v3.21.7

package rpcs

import (
	context "context"
	wsrpc "github.com/goplugin/wsrpc"
)

// EchoClient is the client API for Echo service.
//
type EchoClient interface {
	Echo(ctx context.Context, in *EchoRequest) (*EchoResponse, error)
}

type echoClient struct {
	cc wsrpc.ClientInterface
}

func NewEchoClient(cc wsrpc.ClientInterface) EchoClient {
	return &echoClient{cc}
}

func (c *echoClient) Echo(ctx context.Context, in *EchoRequest) (*EchoResponse, error) {
	out := new(EchoResponse)
	err := c.cc.Invoke(ctx, "Echo", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// EchoServer is the server API for Echo service.
type EchoServer interface {
	Echo(context.Context, *EchoRequest) (*EchoResponse, error)
}

func RegisterEchoServer(s wsrpc.ServiceRegistrar, srv EchoServer) {
	s.RegisterService(&Echo_ServiceDesc, srv)
}

func _Echo_Echo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error) (interface{}, error) {
	in := new(EchoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	return srv.(EchoServer).Echo(ctx, in)
}

// Echo_ServiceDesc is the wsrpc.ServiceDesc for Echo service.
// It's only intended for direct use with wsrpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Echo_ServiceDesc = wsrpc.ServiceDesc{
	ServiceName: "rpcs.Echo",
	HandlerType: (*EchoServer)(nil),
	Methods: []wsrpc.MethodDesc{
		{
			MethodName: "Echo",
			Handler:    _Echo_Echo_Handler,
		},
	},
}

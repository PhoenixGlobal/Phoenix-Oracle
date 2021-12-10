package proto

import (
	context "context"

	wsrpc "github.com/smartcontractkit/wsrpc"
)

type FeedsManagerClient interface {
	ApprovedJob(ctx context.Context, in *ApprovedJobRequest) (*ApprovedJobResponse, error)
	UpdateNode(ctx context.Context, in *UpdateNodeRequest) (*UpdateNodeResponse, error)
	RejectedJob(ctx context.Context, in *RejectedJobRequest) (*RejectedJobResponse, error)
}

type feedsManagerClient struct {
	cc wsrpc.ClientInterface
}

func NewFeedsManagerClient(cc wsrpc.ClientInterface) FeedsManagerClient {
	return &feedsManagerClient{cc}
}

func (c *feedsManagerClient) ApprovedJob(ctx context.Context, in *ApprovedJobRequest) (*ApprovedJobResponse, error) {
	out := new(ApprovedJobResponse)
	err := c.cc.Invoke(ctx, "ApprovedJob", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *feedsManagerClient) UpdateNode(ctx context.Context, in *UpdateNodeRequest) (*UpdateNodeResponse, error) {
	out := new(UpdateNodeResponse)
	err := c.cc.Invoke(ctx, "UpdateNode", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *feedsManagerClient) RejectedJob(ctx context.Context, in *RejectedJobRequest) (*RejectedJobResponse, error) {
	out := new(RejectedJobResponse)
	err := c.cc.Invoke(ctx, "RejectedJob", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// FeedsManagerServer is the server API for FeedsManager service.
type FeedsManagerServer interface {
	ApprovedJob(context.Context, *ApprovedJobRequest) (*ApprovedJobResponse, error)
	UpdateNode(context.Context, *UpdateNodeRequest) (*UpdateNodeResponse, error)
	RejectedJob(context.Context, *RejectedJobRequest) (*RejectedJobResponse, error)
}

func RegisterFeedsManagerServer(s wsrpc.ServiceRegistrar, srv FeedsManagerServer) {
	s.RegisterService(&FeedsManager_ServiceDesc, srv)
}

func _FeedsManager_ApprovedJob_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error) (interface{}, error) {
	in := new(ApprovedJobRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	return srv.(FeedsManagerServer).ApprovedJob(ctx, in)
}

func _FeedsManager_UpdateNode_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error) (interface{}, error) {
	in := new(UpdateNodeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	return srv.(FeedsManagerServer).UpdateNode(ctx, in)
}

func _FeedsManager_RejectedJob_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error) (interface{}, error) {
	in := new(RejectedJobRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	return srv.(FeedsManagerServer).RejectedJob(ctx, in)
}

// FeedsManager_ServiceDesc is the wsrpc.ServiceDesc for FeedsManager service.
// It's only intended for direct use with wsrpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var FeedsManager_ServiceDesc = wsrpc.ServiceDesc{
	ServiceName: "cfm.FeedsManager",
	HandlerType: (*FeedsManagerServer)(nil),
	Methods: []wsrpc.MethodDesc{
		{
			MethodName: "ApprovedJob",
			Handler:    _FeedsManager_ApprovedJob_Handler,
		},
		{
			MethodName: "UpdateNode",
			Handler:    _FeedsManager_UpdateNode_Handler,
		},
		{
			MethodName: "RejectedJob",
			Handler:    _FeedsManager_RejectedJob_Handler,
		},
	},
}

// NodeServiceClient is the client API for NodeService service.
//
type NodeServiceClient interface {
	ProposeJob(ctx context.Context, in *ProposeJobRequest) (*ProposeJobResponse, error)
}

type nodeServiceClient struct {
	cc wsrpc.ClientInterface
}

func NewNodeServiceClient(cc wsrpc.ClientInterface) NodeServiceClient {
	return &nodeServiceClient{cc}
}

func (c *nodeServiceClient) ProposeJob(ctx context.Context, in *ProposeJobRequest) (*ProposeJobResponse, error) {
	out := new(ProposeJobResponse)
	err := c.cc.Invoke(ctx, "ProposeJob", in, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// NodeServiceServer is the server API for NodeService service.
type NodeServiceServer interface {
	ProposeJob(context.Context, *ProposeJobRequest) (*ProposeJobResponse, error)
}

func RegisterNodeServiceServer(s wsrpc.ServiceRegistrar, srv NodeServiceServer) {
	s.RegisterService(&NodeService_ServiceDesc, srv)
}

func _NodeService_ProposeJob_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error) (interface{}, error) {
	in := new(ProposeJobRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	return srv.(NodeServiceServer).ProposeJob(ctx, in)
}

// NodeService_ServiceDesc is the wsrpc.ServiceDesc for NodeService service.
// It's only intended for direct use with wsrpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var NodeService_ServiceDesc = wsrpc.ServiceDesc{
	ServiceName: "cfm.NodeService",
	HandlerType: (*NodeServiceServer)(nil),
	Methods: []wsrpc.MethodDesc{
		{
			MethodName: "ProposeJob",
			Handler:    _NodeService_ProposeJob_Handler,
		},
	},
}

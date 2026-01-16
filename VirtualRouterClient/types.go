package VirtualRouterClient

import (
	"virtual-router-go/internal/core"
	"virtual-router-go/internal/rpc"
)

// 公开类型别名

type RouteMessageType = core.RouteMessageType

type RouteNode = core.RouteNode

type RpcStubMetadata = core.RpcStubMetadata

type RpcParamMeta = rpc.RpcParamMeta

type RpcFuncMeta = rpc.RpcFuncMeta

const (
	RouteMessageTypeHeartBeat       = core.RouteMessageTypeHeartBeat
	RouteMessageTypeMessageData     = core.RouteMessageTypeMessageData
	RouteMessageTypeRemoveRouteNode = core.RouteMessageTypeRemoveRouteNode
	RouteMessageTypeRpcRequest      = core.RouteMessageTypeRpcRequest
	RouteMessageTypeRpcResponse     = core.RouteMessageTypeRpcResponse
	RouteMessageTypeSystemError     = core.RouteMessageTypeSystemError
)

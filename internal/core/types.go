package core

import "fmt"

// RouteMessageType 与 Kotlin EnumRouteMessageType 顺序保持一致
// 0=HeartBeat,1=MessageData,2=RemoveRouteNode,3=RpcRequest,4=RpcResponse,5=SystemError

type RouteMessageType int32

const (
	RouteMessageTypeHeartBeat RouteMessageType = iota
	RouteMessageTypeMessageData
	RouteMessageTypeRemoveRouteNode
	RouteMessageTypeRpcRequest
	RouteMessageTypeRpcResponse
	RouteMessageTypeSystemError
)

func (t RouteMessageType) String() string {
	switch t {
	case RouteMessageTypeHeartBeat:
		return "HeartBeat"
	case RouteMessageTypeMessageData:
		return "MessageData"
	case RouteMessageTypeRemoveRouteNode:
		return "RemoveRouteNode"
	case RouteMessageTypeRpcRequest:
		return "RpcRequest"
	case RouteMessageTypeRpcResponse:
		return "RpcResponse"
	case RouteMessageTypeSystemError:
		return "SystemError"
	default:
		return fmt.Sprintf("Unknown(%d)", int32(t))
	}
}

func RouteMessageTypeFromOrdinal(v int32) (*RouteMessageType, bool) {
	if v < 0 || v > int32(RouteMessageTypeSystemError) {
		return nil, false
	}
	mt := RouteMessageType(v)
	return &mt, true
}

type RouteNode struct {
	RouterId   string `json:"routerId"`
	HostForRpc string `json:"hostForRpc"`
	PortForRpc int    `json:"portForRpc"`
}

type RpcStubMetadata struct {
	PacketId              int      `json:"packetId"`
	Description           string   `json:"description"`
	ClassName             string   `json:"className"`
	MethodName            string   `json:"methodName"`
	ParameterTypes        []string `json:"parameterTypes"`
	ParameterNames        []string `json:"parameterNames"`
	ParameterDescriptions []string `json:"parameterDescriptions"`
	ParameterExampleJson  []string `json:"parameterExampleJson"`
}

type RpcServerInfo struct {
	Host  string            `json:"host"`
	Port  int               `json:"port"`
	Stubs []RpcStubMetadata `json:"stubs"`
}

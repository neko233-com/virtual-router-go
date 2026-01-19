package rpc

import (
	"encoding/json"
	"errors"
	"sync/atomic"
	"time"

	"github.com/neko233-com/virtual-router-go/internal/core"
)

type RelayClient struct {
	targetRouteId string
	routerClient  RouterClientSender
}

func NewRelayClient(targetRouteId string, client RouterClientSender) *RelayClient {
	return &RelayClient{targetRouteId: targetRouteId, routerClient: client}
}

func (c *RelayClient) Call(packetId int, timeout time.Duration, args []json.RawMessage) (string, error) {
	if !c.routerClient.IsConnected() {
		return "", errors.New("VirtualRouterClient 未连接，无法发起 RPC 调用")
	}

	// 本地调用优化
	if c.targetRouteId == c.routerClient.RouteId() {
		return invokeLocal(packetId, args)
	}

	req := &RpcRequest{
		FromRouteId:        c.routerClient.RouteId(),
		ToRouteId:          c.targetRouteId,
		RpcUid:             GenerateRpcUid(),
		StartTimeMs:        time.Now().UnixMilli(),
		PacketId:           packetId,
		MethodArgsJsonList: rawToStringList(args),
	}

	future := NewFuture(req.RpcUid)
	RelayFutureManagerInstance().Register(future)
	_ = c.routerClient.Send(c.targetRouteId, core.RouteMessageTypeRpcRequest, req)
	return future.Await(timeout)
}

func invokeLocal(packetId int, args []json.RawMessage) (string, error) {
	result, err := ServerStubManagerInstance().Invoke(packetId, args)
	if err != nil {
		return "", err
	}
	return toJsonOrString(result), nil
}

var uidCounter atomic.Int64

func GenerateRpcUid() string {
	return "relay-" + int64ToString(time.Now().UnixMilli()) + "-" + int64ToString(uidCounter.Add(1))
}

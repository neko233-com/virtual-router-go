package rpc

import (
	"encoding/json"
	"log"

	"github.com/neko233-com/virtual-router-go/internal/core"
)

var relayFutureManager = NewFutureManager()
var waitResultManager = NewFutureManager()

func RelayFutureManagerInstance() *FutureManager {
	return relayFutureManager
}

func WaitResultManagerInstance() *FutureManager {
	return waitResultManager
}

func HandleRelayRpcRequest(msg *core.RouteMessage, client RouterClientSender) {
	if msg.Data == nil {
		return
	}
	var req RpcRequest
	if err := json.Unmarshal([]byte(*msg.Data), &req); err != nil {
		log.Printf("RPC 请求解析失败: %v", err)
		return
	}

	resp := RpcResponse{RpcUid: req.RpcUid, StartTimeMs: req.StartTimeMs, PacketId: req.PacketId}
	result, err := ServerStubManagerInstance().Invoke(req.PacketId, rawToJsonArgs(req.MethodArgsJsonList))
	if err != nil {
		resp.ErrorFlag = true
		resp.ErrorMsg = err.Error()
	} else {
		resp.ResultValueStr = toJsonOrString(result)
	}

	_ = client.Send(msg.FromRouteId, core.RouteMessageTypeRpcResponse, resp)
}

func HandleRelayRpcResponse(msg *core.RouteMessage) {
	if msg.Data == nil {
		return
	}
	var resp RpcResponse
	if err := json.Unmarshal([]byte(*msg.Data), &resp); err != nil {
		log.Printf("RPC 响应解析失败: %v", err)
		return
	}
	if resp.ErrorFlag {
		RelayFutureManagerInstance().SetError(resp.RpcUid, resp.ErrorMsg)
	} else {
		RelayFutureManagerInstance().SetSuccess(resp.RpcUid, resp.ResultValueStr)
	}
}

func rawToStringList(args []json.RawMessage) []string {
	list := make([]string, 0, len(args))
	for _, a := range args {
		list = append(list, string(a))
	}
	return list
}

func rawToJsonArgs(args []string) []json.RawMessage {
	list := make([]json.RawMessage, 0, len(args))
	for _, a := range args {
		list = append(list, json.RawMessage([]byte(a)))
	}
	return list
}

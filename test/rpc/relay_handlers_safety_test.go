package rpc_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/neko233-com/virtual-router-go/internal/core"
	"github.com/neko233-com/virtual-router-go/internal/rpc"
)

type panicSender struct{}

func (p *panicSender) Send(toRouteId string, msgType core.RouteMessageType, obj any) error {
	panic("send boom")
}
func (p *panicSender) IsConnected() bool { return true }
func (p *panicSender) RouteId() string   { return "sender" }
func (p *panicSender) AwaitConnected(timeout time.Duration) error {
	return nil
}

type captureSender struct {
	lastObj any
}

func (c *captureSender) Send(toRouteId string, msgType core.RouteMessageType, obj any) error {
	c.lastObj = obj
	return nil
}
func (c *captureSender) IsConnected() bool { return true }
func (c *captureSender) RouteId() string   { return "sender" }
func (c *captureSender) AwaitConnected(timeout time.Duration) error {
	return nil
}

func TestHandleRelayRpcRequest_SendPanicShouldNotCrash(t *testing.T) {
	rpc.ServerStubManagerInstance().Reset()
	defer rpc.ServerStubManagerInstance().Reset()

	if err := rpc.RegisterRpcFunc(rpc.RpcFuncMeta{PacketId: 70001, Description: "ok"}, func() (string, error) {
		return "ok", nil
	}); err != nil {
		t.Fatalf("register stub error: %v", err)
	}

	req := map[string]any{
		"rpcUid":             "u1",
		"packetId":           70001,
		"methodArgsJsonList": []string{},
	}
	b, _ := json.Marshal(req)
	data := string(b)
	mt := core.RouteMessageTypeRpcRequest
	msg := &core.RouteMessage{FromRouteId: "a", ToRouteId: "b", MessageType: &mt, Data: &data}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("HandleRelayRpcRequest should recover internally, panic=%v", r)
		}
	}()
	rpc.HandleRelayRpcRequest(msg, &panicSender{})
}

func TestHandleRelayRpcRequest_BusinessPanicShouldReturnErrorResponse(t *testing.T) {
	rpc.ServerStubManagerInstance().Reset()
	defer rpc.ServerStubManagerInstance().Reset()

	rpc.ServerStubManagerInstance().RegisterStub(core.RpcStubMetadata{PacketId: 70002, Description: "panic"}, func(args []json.RawMessage) (any, error) {
		panic("handler panic")
	})

	req := map[string]any{
		"rpcUid":             "u2",
		"packetId":           70002,
		"methodArgsJsonList": []string{},
	}
	b, _ := json.Marshal(req)
	data := string(b)
	mt := core.RouteMessageTypeRpcRequest
	msg := &core.RouteMessage{FromRouteId: "a", ToRouteId: "b", MessageType: &mt, Data: &data}

	s := &captureSender{}
	rpc.HandleRelayRpcRequest(msg, s)

	resp, ok := s.lastObj.(rpc.RpcResponse)
	if !ok {
		t.Fatalf("expected RpcResponse, got %#v", s.lastObj)
	}
	if !resp.ErrorFlag {
		t.Fatal("expected ErrorFlag=true when handler panics")
	}
	if resp.ErrorMsg == "" {
		t.Fatal("expected error message when handler panics")
	}
	if !strings.Contains(resp.ErrorMsg, "panic") {
		t.Fatalf("expected panic keyword in error, got %q", resp.ErrorMsg)
	}
}

func TestHandleRelayRpcResponse_ShouldNotCrashOnBadFutureState(t *testing.T) {
	// 无注册 future 时应该直接忽略，不应 panic。
	msgData := `{"rpcUid":"not-exists","errorFlag":true,"errorMsg":"x"}`
	mt := core.RouteMessageTypeRpcResponse
	msg := &core.RouteMessage{FromRouteId: "a", ToRouteId: "b", MessageType: &mt, Data: &msgData}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("HandleRelayRpcResponse should not panic, got=%v", r)
		}
	}()
	rpc.HandleRelayRpcResponse(msg)
}



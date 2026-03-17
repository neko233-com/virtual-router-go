package virtual_router_client_test

import (
	"testing"

	clientpkg "github.com/neko233-com/virtual-router-go/internal/VirtualRouterClient"
	"github.com/neko233-com/virtual-router-go/internal/config"
	"github.com/neko233-com/virtual-router-go/internal/rpc"
)

func TestClientStartReturnsErrorWhenStubNotRegistered(t *testing.T) {
	clientpkg.ResetRouteTableForTest()
	t.Cleanup(clientpkg.ResetRouteTableForTest)

	// 清理全局 Stub 注册状态，模拟业务未注册 RPC 方法的启动场景。
	rpc.ServerStubManagerInstance().Reset()
	defer rpc.ServerStubManagerInstance().Reset()

	cfg := &config.RouterClientConfig{
		RouteId:                 "ut-client",
		RouterCenterHost:        "127.0.0.1",
		RouterCenterPort:        65534,
		RpcMode:                 "relay",
		HeartBeatIntervalSecond: 1,
		ReconnectIntervalMs:     100,
	}
	client := clientpkg.NewClientByConfig(cfg)
	if client == nil {
		t.Fatal("NewClientByConfig 返回 nil")
	}
	defer client.Shutdown()

	if err := client.Start(); err == nil {
		t.Fatal("期望启动阶段返回错误，但实际为 nil")
	}
}




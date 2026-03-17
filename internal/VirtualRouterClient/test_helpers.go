package VirtualRouterClient

import (
	"github.com/neko233-com/virtual-router-go/internal/core"
	"github.com/neko233-com/virtual-router-go/internal/rpc"
)

// ResetRouteTableForTest 重置全局路由表单例，避免测试之间共享状态导致误判。
func ResetRouteTableForTest() {
	t := RouteTableInstance()
	t.mu.Lock()
	defer t.mu.Unlock()

	t.routeId = ""
	t.routerClient = nil
	t.rpcMode = ""

	for _, c := range t.routeIdToRpcClient {
		if c != nil {
			c.Close()
		}
	}
	t.routeIdToNodeMap = map[string]core.RouteNode{}
	t.routeIdToRpcClient = map[string]*rpc.DirectClient{}
	t.routeIdToRelay = map[string]*rpc.RelayClient{}
}

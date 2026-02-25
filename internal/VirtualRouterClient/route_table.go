package VirtualRouterClient

import (
	"log/slog"
	"strings"
	"sync"

	"github.com/neko233-com/virtual-router-go/internal/core"
	"github.com/neko233-com/virtual-router-go/internal/rpc"
)

type RouteTable struct {
	routeId      string
	routerClient *Client
	rpcMode      string

	mu                 sync.RWMutex
	routeIdToNodeMap   map[string]core.RouteNode
	routeIdToRpcClient map[string]*rpc.DirectClient
	routeIdToRelay     map[string]*rpc.RelayClient
}

var routeTableInstance = &RouteTable{
	routeIdToNodeMap:   map[string]core.RouteNode{},
	routeIdToRpcClient: map[string]*rpc.DirectClient{},
	routeIdToRelay:     map[string]*rpc.RelayClient{},
}

func RouteTableInstance() *RouteTable {
	return routeTableInstance
}

func (t *RouteTable) SetRouteId(routeId string) {
	if t.routeId != "" {
		panic("不允许覆盖 routeId")
	}
	t.routeId = routeId
}

func (t *RouteTable) RouteId() string {
	return t.routeId
}

func (t *RouteTable) SetRouterClient(c *Client) {
	t.routerClient = c
}

func (t *RouteTable) SetRpcMode(mode string) {
	if mode == "" {
		mode = "relay"
	}
	t.rpcMode = mode
}

func (t *RouteTable) UpsertRouteNode(nodes []core.RouteNode) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, node := range nodes {
		old, ok := t.routeIdToNodeMap[node.RouterId]
		if ok && old == node {
			continue
		}
		if ok {
			// 连接信息变更，清理旧连接
			delete(t.routeIdToRpcClient, node.RouterId)
			slog.Info("路由连接信息变更，关闭历史连接", "routeId", node.RouterId)
		}
		t.routeIdToNodeMap[node.RouterId] = node
	}
}

func (t *RouteTable) RemoveRouteNode(routeIds []string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, rid := range routeIds {
		delete(t.routeIdToNodeMap, rid)
		if client, ok := t.routeIdToRpcClient[rid]; ok {
			client.Close()
			delete(t.routeIdToRpcClient, rid)
		}
	}
}

func (t *RouteTable) GetOrCreateRpcClient(routeId string) (*rpc.DirectClient, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if c, ok := t.routeIdToRpcClient[routeId]; ok {
		return c, nil
	}
	routeNode, ok := t.routeIdToNodeMap[routeId]
	if !ok {
		return nil, rpc.ErrRouteNotFound
	}
	client := rpc.NewDirectClient(t.routeId, routeId, routeNode.HostForRpc, routeNode.PortForRpc)
	go client.Start()
	t.routeIdToRpcClient[routeId] = client
	return client, nil
}

func (t *RouteTable) GetRpcServiceProvider(routeId string) (rpc.ServiceProvider, error) {
	if strings.EqualFold(t.rpcMode, "relay") {
		return t.getOrCreateRelay(routeId)
	}
	return t.GetOrCreateRpcClient(routeId)
}

func (t *RouteTable) getOrCreateRelay(routeId string) (*rpc.RelayClient, error) {
	if t.routerClient == nil {
		return nil, rpc.ErrRouterClientRequired
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if c, ok := t.routeIdToRelay[routeId]; ok {
		return c, nil
	}
	client := rpc.NewRelayClient(routeId, t.routerClient)
	t.routeIdToRelay[routeId] = client
	return client, nil
}

func (t *RouteTable) HasAnyRouteNode() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.routeIdToNodeMap) > 0
}

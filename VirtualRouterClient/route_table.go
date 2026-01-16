package VirtualRouterClient

import (
	"encoding/json"
	"time"

	internalClient "virtual-router-go/internal/VirtualRouterClient"
	"virtual-router-go/internal/rpc"
)

type ServiceProvider interface {
	Call(packetId int, timeout time.Duration, args []json.RawMessage) (string, error)
}

type serviceProviderAdapter struct {
	inner rpc.ServiceProvider
}

func (s *serviceProviderAdapter) Call(packetId int, timeout time.Duration, args []json.RawMessage) (string, error) {
	return s.inner.Call(packetId, timeout, args)
}

type RouteTable struct {
	inner *internalClient.RouteTable
}

func RouteTableInstance() *RouteTable {
	return &RouteTable{inner: internalClient.RouteTableInstance()}
}

func (t *RouteTable) GetRpcServiceProvider(toRouteId string) (ServiceProvider, error) {
	provider, err := t.inner.GetRpcServiceProvider(toRouteId)
	if err != nil {
		return nil, err
	}
	return &serviceProviderAdapter{inner: provider}, nil
}

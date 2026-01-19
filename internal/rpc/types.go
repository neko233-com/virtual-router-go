package rpc

import (
	"errors"
	"github.com/neko233-com/virtual-router-go/internal/core"
)

var (
	ErrRouteNotFound        = errors.New("路由节点未注册到路由中心")
	ErrRouterClientRequired = errors.New("Relay 模式需要 VirtualRouterClient")
	ErrFrameTooLarge        = errors.New("frame length out of range")
)

type RouterClientSender interface {
	Send(toRouteId string, msgType core.RouteMessageType, obj any) error
	IsConnected() bool
	RouteId() string
}

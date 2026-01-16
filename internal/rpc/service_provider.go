package rpc

import (
	"encoding/json"
	"time"
)

// ServiceProvider 统一调用接口

type ServiceProvider interface {
	Call(packetId int, timeout time.Duration, args []json.RawMessage) (string, error)
}

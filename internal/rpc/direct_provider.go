package rpc

import (
	"encoding/json"
	"time"
)

func (c *DirectClient) Call(packetId int, timeout time.Duration, args []json.RawMessage) (string, error) {
	return c.GetOrCreateProxy(packetId, timeout, args)
}

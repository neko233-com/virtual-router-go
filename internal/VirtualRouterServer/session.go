package VirtualRouterServer

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	"virtual-router-go/internal/core"
)

type RouterSession struct {
	RouterId      string
	Conn          net.Conn
	RpcServerInfo core.RpcServerInfo
	lastHeartbeat atomic.Int64
	closed        atomic.Bool
	writeMu       *sync.Mutex
}

func NewRouterSession(routeId string, conn net.Conn, info core.RpcServerInfo, writeMu *sync.Mutex) *RouterSession {
	s := &RouterSession{
		RouterId:      routeId,
		Conn:          conn,
		RpcServerInfo: info,
		writeMu:       writeMu,
	}
	s.RefreshHeartbeat()
	return s
}

func (s *RouterSession) RefreshHeartbeat() {
	s.lastHeartbeat.Store(time.Now().UnixMilli())
}

func (s *RouterSession) LastHeartbeatMs() int64 {
	return s.lastHeartbeat.Load()
}

func (s *RouterSession) IsActive() bool {
	return !s.closed.Load()
}

func (s *RouterSession) MarkClosed() {
	s.closed.Store(true)
}

func (s *RouterSession) WriteRouteMessage(msg *core.RouteMessage) error {
	payload, err := msg.EncodePayload()
	if err != nil {
		return err
	}
	frame := core.EncodeFrame(payload)
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err = s.Conn.Write(frame)
	return err
}

func (s *RouterSession) WritePayload(payload []byte) error {
	frame := core.EncodeFrame(payload)
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err := s.Conn.Write(frame)
	return err
}

func (s *RouterSession) RemoteAddrStr() string {
	if s.Conn == nil {
		return ""
	}
	return s.Conn.RemoteAddr().String()
}

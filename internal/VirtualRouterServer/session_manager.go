package VirtualRouterServer

import (
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/neko233-com/virtual-router-go/internal/core"
)

const sessionTimeout = 30 * time.Second

// RouterSessionManager 管理所有路由会话

type RouterSessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*RouterSession
}

func NewRouterSessionManager() *RouterSessionManager {
	sm := &RouterSessionManager{
		sessions: make(map[string]*RouterSession),
	}
	go sm.heartbeatLoop()
	return sm
}

func (m *RouterSessionManager) heartbeatLoop() {
	for {
		time.Sleep(time.Minute)
		m.checkHeartbeat()
	}
}

func (m *RouterSessionManager) checkHeartbeat() {
	cutoff := time.Now().Add(-sessionTimeout).UnixMilli()
	var offline []string

	m.mu.RLock()
	for routeId, session := range m.sessions {
		if session.LastHeartbeatMs() < cutoff {
			offline = append(offline, routeId)
		}
	}
	m.mu.RUnlock()

	if len(offline) > 0 {
		m.RemoveSessions(offline)
	}
}

func (m *RouterSessionManager) UpsertSession(routeId string, session *RouterSession) (*RouterSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if old, ok := m.sessions[routeId]; ok {
		if old.IsActive() {
			if old.RemoteAddrStr() == session.RemoteAddrStr() {
				old.RefreshHeartbeat()
				return old, nil
			}
			return nil, errors.New("RouterId '" + routeId + "' 已经存在! 请修改您的 routerId 配置.")
		}
	}
	m.sessions[routeId] = session
	return session, nil
}

func (m *RouterSessionManager) RemoveSession(routeId string) {
	m.RemoveSessions([]string{routeId})
}

func (m *RouterSessionManager) RemoveSessions(routeIds []string) {
	var removed []string
	m.mu.Lock()
	for _, routeId := range routeIds {
		session, ok := m.sessions[routeId]
		if !ok {
			continue
		}
		delete(m.sessions, routeId)
		removed = append(removed, routeId)
		session.MarkClosed()
		_ = session.Conn.Close()
		slog.Info("client 的 routeSession 被移除了", "routeId", routeId, "remote", session.RemoteAddrStr())
	}
	m.mu.Unlock()

	if len(removed) == 0 {
		return
	}

	// 通知剩余活着的节点
	m.mu.RLock()
	defer m.mu.RUnlock()
	dataBytes, _ := jsonMarshal(removed)
	dataStr := string(dataBytes)
	msgType := core.RouteMessageTypeRemoveRouteNode
	for _, alive := range m.sessions {
		msg := &core.RouteMessage{
			FromRouteId: alive.RouterId,
			ToRouteId:   alive.RouterId,
			MessageType: &msgType,
			Data:        &dataStr,
		}
		_ = alive.WriteRouteMessage(msg)
	}
}

func (m *RouterSessionManager) GetSession(routeId string) *RouterSession {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sessions[routeId]
}

func (m *RouterSessionManager) GetAllRouteNodeList() []core.RouteNode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]core.RouteNode, 0, len(m.sessions))
	for _, s := range m.sessions {
		list = append(list, core.RouteNode{
			RouterId:   s.RouterId,
			HostForRpc: s.RpcServerInfo.Host,
			PortForRpc: s.RpcServerInfo.Port,
		})
	}
	return list
}

func (m *RouterSessionManager) RefreshSession(routeId string) {
	m.mu.RLock()
	s := m.sessions[routeId]
	m.mu.RUnlock()
	if s != nil {
		s.RefreshHeartbeat()
	}
}

func jsonMarshal(v any) ([]byte, error) {
	return jsonMarshalImpl(v)
}

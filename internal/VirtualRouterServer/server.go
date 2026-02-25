package VirtualRouterServer

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/neko233-com/virtual-router-go/internal/config"
	"github.com/neko233-com/virtual-router-go/internal/core"
)

type Server struct {
	cfg                *config.RouterServerConfig
	sessionManager     *RouterSessionManager
	listener           net.Listener
	startTime          time.Time
	totalBytes         atomic.Uint64
	totalRequests      atomic.Uint64
	totalConnections   atomic.Uint64
	currentConnections atomic.Int64
	debugResults       sync.Map
	shutdownCh         chan struct{}
}

func NewServer(cfg *config.RouterServerConfig) *Server {
	return &Server{
		cfg:            cfg,
		sessionManager: NewRouterSessionManager(),
		startTime:      time.Now(),
		shutdownCh:     make(chan struct{}),
	}
}

func (s *Server) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", ":"+intToString(s.cfg.RouterServerPort))
	if err != nil {
		return err
	}
	s.listener = ln
	log.Printf("Router Server 启动成功, 端口=%d", s.cfg.RouterServerPort)
	logTCPAccessAddresses("Router Server", s.cfg.RouterServerPort)

	go func() {
		<-ctx.Done()
		_ = s.Shutdown()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-s.shutdownCh:
				return nil
			default:
				log.Printf("accept error: %v", err)
				continue
			}
		}
		s.totalConnections.Add(1)
		s.currentConnections.Add(1)
		go s.handleConn(conn)
	}
}

func (s *Server) Shutdown() error {
	select {
	case <-s.shutdownCh:
		return nil
	default:
		close(s.shutdownCh)
	}
	if s.listener != nil {
		_ = s.listener.Close()
	}
	return nil
}

func (s *Server) handleConn(conn net.Conn) {
	defer func() {
		s.currentConnections.Add(-1)
		_ = conn.Close()
	}()

	var routeId string
	writeMu := &sync.Mutex{}

	for {
		payload, err := core.ReadFrame(conn)
		if err != nil {
			if routeId != "" {
				s.sessionManager.RemoveSession(routeId)
			}
			return
		}
		s.totalRequests.Add(1)
		s.totalBytes.Add(uint64(len(payload)))

		msg, err := core.DecodeRouteMessagePayload(payload)
		if err != nil {
			log.Printf("decode route message error: %v", err)
			continue
		}

		if msg.MessageType == nil {
			log.Printf("msgType=null remote=%s from=%s to=%s", conn.RemoteAddr().String(), msg.FromRouteId, msg.ToRouteId)
			continue
		}

		if msg.FromRouteId != "" {
			routeId = msg.FromRouteId
		}

		s.handleRouteMessage(msg, conn, writeMu)
	}
}

func (s *Server) handleRouteMessage(msg *core.RouteMessage, conn net.Conn, writeMu *sync.Mutex) {
	switch *msg.MessageType {
	case core.RouteMessageTypeHeartBeat:
		s.handleHeartBeat(msg, conn, writeMu)
	case core.RouteMessageTypeMessageData:
		s.forwardToTarget(msg)
	case core.RouteMessageTypeRpcRequest:
		s.forwardToTarget(msg)
	case core.RouteMessageTypeRpcResponse:
		s.handleRpcResponse(msg)
	case core.RouteMessageTypeSystemError:
		return
	case core.RouteMessageTypeRemoveRouteNode:
		return
	default:
		return
	}
}

func (s *Server) handleHeartBeat(msg *core.RouteMessage, conn net.Conn, writeMu *sync.Mutex) {
	if msg.Data == nil {
		return
	}
	var rpcInfo core.RpcServerInfo
	if err := json.Unmarshal([]byte(*msg.Data), &rpcInfo); err != nil {
		log.Printf("heartbeat parse error: %v", err)
		return
	}

	newSession := NewRouterSession(msg.FromRouteId, conn, rpcInfo, writeMu)
	newSession.RefreshHeartbeat()

	session, err := s.sessionManager.UpsertSession(msg.FromRouteId, newSession)
	if err != nil {
		// RouterId 冲突
		errorMsg := err.Error()
		mt := core.RouteMessageTypeSystemError
		resp := &core.RouteMessage{
			FromRouteId: "server",
			ToRouteId:   msg.FromRouteId,
			MessageType: &mt,
			Data:        &errorMsg,
		}
		_ = newSession.WriteRouteMessage(resp)
		_ = conn.Close()
		return
	}

	// 返回路由表
	routeList := s.sessionManager.GetAllRouteNodeList()
	jsonBytes, _ := json.Marshal(routeList)
	jsonStr := string(jsonBytes)
	respMsg := &core.RouteMessage{
		FromRouteId: msg.FromRouteId,
		ToRouteId:   msg.FromRouteId,
		MessageType: msg.MessageType,
		Data:        &jsonStr,
	}
	_ = session.WriteRouteMessage(respMsg)
}

func (s *Server) forwardToTarget(msg *core.RouteMessage) {
	if msg.ToRouteId == "" {
		return
	}
	target := s.sessionManager.GetSession(msg.ToRouteId)
	if target == nil {
		log.Printf("route message error! to routeId offline. from=%s to=%s type=%s", msg.FromRouteId, msg.ToRouteId, msg.MessageType.String())
		return
	}
	_ = target.WriteRouteMessage(msg)
}

func (s *Server) handleRpcResponse(msg *core.RouteMessage) {
	if msg.ToRouteId == "debug-admin" {
		if msg.Data == nil {
			return
		}
		rpcUid := extractRpcUid(*msg.Data)
		if rpcUid == "" {
			return
		}
		s.debugResults.Store(rpcUid, *msg.Data)
		go func(uid string) {
			time.Sleep(5 * time.Minute)
			s.debugResults.Delete(uid)
		}(rpcUid)
		return
	}

	s.forwardToTarget(msg)
}

func (s *Server) GetDebugResult(rpcUid string) (string, bool) {
	if v, ok := s.debugResults.Load(rpcUid); ok {
		return v.(string), true
	}
	return "", false
}

func (s *Server) SessionManager() *RouterSessionManager {
	return s.sessionManager
}

func (s *Server) Stats() (totalConn uint64, currentConn int64, totalBytes uint64, totalRequests uint64, uptimeMs int64) {
	return s.totalConnections.Load(), s.currentConnections.Load(), s.totalBytes.Load(), s.totalRequests.Load(), time.Since(s.startTime).Milliseconds()
}

var rpcUidRegex = regexp.MustCompile(`"rpcUid"\s*:\s*"([^"]+)"`)

func extractRpcUid(jsonStr string) string {
	match := rpcUidRegex.FindStringSubmatch(jsonStr)
	if len(match) >= 2 {
		return match[1]
	}
	// 兼容数字 rpcUid
	if strings.Contains(jsonStr, "rpcUid") {
		var temp struct {
			RpcUid any `json:"rpcUid"`
		}
		if err := json.Unmarshal([]byte(jsonStr), &temp); err == nil {
			return toString(temp.RpcUid)
		}
	}
	return ""
}

func toString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return intToString(int(t))
	case int:
		return intToString(t)
	case int64:
		return intToString(int(t))
	default:
		return ""
	}
}

func intToString(v int) string {
	return strconv.Itoa(v)
}

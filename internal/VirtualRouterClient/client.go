package VirtualRouterClient

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/neko233-com/virtual-router-go/internal/config"
	"github.com/neko233-com/virtual-router-go/internal/core"
	"github.com/neko233-com/virtual-router-go/internal/rpc"
)

type Client struct {
	cfg              *config.RouterClientConfig
	routeId          string
	routerCenterHost string
	routerCenterPort int
	conn             net.Conn
	writeMu          sync.Mutex
	needConnect      atomic.Bool
	isOpen           atomic.Bool
	reconnectAttempt atomic.Bool
	stopCh           chan struct{}
}

func NewClient(configFile string) (*Client, error) {
	cfg, err := config.ReadRouterClientConfig(configFile)
	if err != nil {
		return nil, err
	}
	return NewClientByConfig(cfg), nil
}

func NewClientByConfig(cfg *config.RouterClientConfig) *Client {
	if cfg == nil {
		return nil
	}
	c := &Client{
		cfg:              cfg,
		routeId:          cfg.RouteId,
		routerCenterHost: cfg.RouterCenterHost,
		routerCenterPort: cfg.RouterCenterPort,
		stopCh:           make(chan struct{}),
	}

	RouteTableInstance().SetRouteId(c.routeId)
	RouteTableInstance().SetRpcMode(cfg.RpcMode)
	RouteTableInstance().SetRouterClient(c)

	slog.Info("RPC 模式", "mode", strings.ToUpper(cfg.RpcMode))
	return c
}

func (c *Client) RouteId() string {
	return c.routeId
}

func (c *Client) Start() error {
	rpc.ServerStubManagerInstance().EnsureInitialized()

	if !c.needConnect.CompareAndSwap(false, true) {
		return nil
	}

	c.runRouterClient()
	c.runRpcServer()
	return nil
}

func (c *Client) runRpcServer() {
	if strings.EqualFold(c.cfg.RpcMode, "direct") {
		slog.Info("RPC 模式: DIRECT，启动本地 RPC 服务器", "port", c.cfg.LocalRpcPort)
		server := rpc.NewStubServer(c.cfg.LocalRpcPort)
		go server.Start()
	} else {
		slog.Info("RPC 模式: RELAY，RPC 调用将通过 Router Center 转发")
	}
}

func (c *Client) runRouterClient() {
	if c.tryConnect() {
		slog.Info("连接 Router Center 成功", "host", c.routerCenterHost, "port", c.routerCenterPort, "routeId", c.routeId)
		c.isOpen.Store(true)
		c.startHeartbeat()
		go c.readLoop()
		return
	}
	slog.Warn("首次连接 Router Center 失败，将在后台自动重连", "host", c.routerCenterHost, "port", c.routerCenterPort)
	c.startBackgroundReconnect()
}

func (c *Client) tryConnect() bool {
	conn, err := net.Dial("tcp", net.JoinHostPort(c.routerCenterHost, intToString(c.routerCenterPort)))
	if err != nil {
		return false
	}
	c.closeConn()
	c.conn = conn
	return true
}

func (c *Client) startHeartbeat() {
	_ = c.sendHeartbeat()
	go func() {
		for {
			if !c.isOpen.Load() {
				return
			}
			select {
			case <-c.stopCh:
				return
			case <-time.After(time.Duration(c.cfg.HeartBeatIntervalSecond) * time.Second):
			}
			if ok := c.sendHeartbeat(); !ok {
				c.onConnectionLost("heartbeat failed", nil)
				return
			}
		}
	}()
}

func (c *Client) sendHeartbeat() bool {
	if c.conn == nil {
		return false
	}
	isDirect := strings.EqualFold(c.cfg.RpcMode, "direct")
	rpcHost := ""
	rpcPort := 0
	if isDirect {
		rpcHost = c.cfg.LocalRpcHost
		rpcPort = c.cfg.LocalRpcPort
	}
	info := core.RpcServerInfo{Host: rpcHost, Port: rpcPort, Stubs: rpc.ServerStubManagerInstance().GetAllStubsMetadata()}
	b, _ := json.Marshal(info)
	data := string(b)
	mt := core.RouteMessageTypeHeartBeat
	msg := &core.RouteMessage{
		FromRouteId: c.routeId,
		ToRouteId:   "",
		MessageType: &mt,
		Data:        &data,
	}
	payload, err := msg.EncodePayload()
	if err != nil {
		return false
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_, err = c.conn.Write(core.EncodeFrame(payload))
	return err == nil
}

func (c *Client) readLoop() {
	for {
		payload, err := core.ReadFrame(c.conn)
		if err != nil {
			c.onConnectionLost("read loop closed", err)
			return
		}
		msg, err := core.DecodeRouteMessagePayload(payload)
		if err != nil || msg.MessageType == nil {
			continue
		}
		c.handleMessage(msg)
	}
}

func (c *Client) handleMessage(msg *core.RouteMessage) {
	switch *msg.MessageType {
	case core.RouteMessageTypeHeartBeat:
		c.handleRegister(msg)
	case core.RouteMessageTypeRemoveRouteNode:
		c.handleRemoveOffline(msg)
	case core.RouteMessageTypeMessageData:
		slog.Info("收到 message data", "data", safeData(msg))
	case core.RouteMessageTypeRpcRequest:
		rpc.HandleRelayRpcRequest(msg, c)
	case core.RouteMessageTypeRpcResponse:
		rpc.HandleRelayRpcResponse(msg)
	case core.RouteMessageTypeSystemError:
		c.handleSystemError(msg)
	}
}

func (c *Client) handleRegister(msg *core.RouteMessage) {
	if msg.Data == nil {
		return
	}
	var nodes []core.RouteNode
	if err := json.Unmarshal([]byte(*msg.Data), &nodes); err != nil {
		slog.Warn("init route info error", "error", err)
		return
	}
	RouteTableInstance().UpsertRouteNode(nodes)
}

func (c *Client) handleRemoveOffline(msg *core.RouteMessage) {
	if msg.Data == nil {
		return
	}
	var ids []string
	if err := json.Unmarshal([]byte(*msg.Data), &ids); err != nil {
		slog.Warn("remove offline parse error", "error", err)
		return
	}
	if len(ids) == 0 {
		return
	}
	RouteTableInstance().RemoveRouteNode(ids)
	slog.Info("删除已离线的 Route Client", "routeIds", ids)
}

func (c *Client) handleSystemError(msg *core.RouteMessage) {
	if msg.Data == nil {
		return
	}
	errMsg := *msg.Data
	slog.Error("收到系统错误", "errorMessage", errMsg)
	if strings.Contains(errMsg, "RouterId") && strings.Contains(errMsg, "已经存在") {
		slog.Error("FATAL ERROR: RouterId 冲突", "detail", errMsg, "hint", "请修改配置文件中的 routeId，然后重启程序")
		panic(errMsg)
	}
}

func (c *Client) startBackgroundReconnect() {
	if !c.needConnect.Load() {
		return
	}
	if !c.reconnectAttempt.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer c.reconnectAttempt.Store(false)
		attempt := 0
		for {
			if !c.needConnect.Load() {
				return
			}
			if !c.isOpen.Load() {
				if c.tryConnect() {
					c.isOpen.Store(true)
					slog.Info("重连 Router Center 成功", "host", c.routerCenterHost, "port", c.routerCenterPort, "routeId", c.routeId)
					c.startHeartbeat()
					go c.readLoop()
					return
				}
				attempt++
			}
			retryInterval := c.nextReconnectDelay(attempt)
			select {
			case <-c.stopCh:
				return
			case <-time.After(retryInterval):
			}
		}
	}()
}

func (c *Client) Send(toRouteId string, msgType core.RouteMessageType, obj any) error {
	if !c.IsConnected() {
		return errors.New("VirtualRouterClient 未连接到 Router Center，无法发送消息")
	}
	b, _ := json.Marshal(obj)
	data := string(b)
	mt := msgType
	msg := &core.RouteMessage{
		FromRouteId: c.routeId,
		ToRouteId:   toRouteId,
		MessageType: &mt,
		Data:        &data,
	}

	if toRouteId == c.routeId {
		c.handleMessage(msg)
		return nil
	}
	payload, err := msg.EncodePayload()
	if err != nil {
		return err
	}
	var conn net.Conn
	c.writeMu.Lock()
	conn = c.conn
	if conn == nil {
		c.writeMu.Unlock()
		return errors.New("VirtualRouterClient 未连接到 Router Center，无法发送消息")
	}
	_, err = conn.Write(core.EncodeFrame(payload))
	c.writeMu.Unlock()
	if err != nil {
		c.onConnectionLost("send failed", err)
	}
	return err
}

func (c *Client) IsConnected() bool {
	return c.isOpen.Load() && c.conn != nil
}

func (c *Client) Shutdown() {
	c.needConnect.Store(false)
	c.isOpen.Store(false)
	close(c.stopCh)
	c.closeConn()
}

func (c *Client) onConnectionLost(reason string, err error) {
	if !c.needConnect.Load() {
		return
	}
	wasOpen := c.isOpen.Swap(false)
	c.closeConn()
	if wasOpen {
		if err != nil {
			slog.Warn("Router Center 连接断开，准备重连", "reason", reason, "error", err, "host", c.routerCenterHost, "port", c.routerCenterPort)
		} else {
			slog.Warn("Router Center 连接断开，准备重连", "reason", reason, "host", c.routerCenterHost, "port", c.routerCenterPort)
		}
	}
	c.startBackgroundReconnect()
}

func (c *Client) closeConn() {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}

func (c *Client) AwaitRpcRouterInfoFirstReady() error {
	count := 0
	maxWait := 100
	start := time.Now()
	for {
		if RouteTableInstance().HasAnyRouteNode() {
			break
		}
		time.Sleep(100 * time.Millisecond)
		count++
		if count > maxWait {
			return errors.New("10s 还是没有收到 router-server 返回任何注册信息, 请检查你的配置")
		}
	}
	slog.Info("等待 router-server 返回 rpc 信息完成", "costMs", time.Since(start).Milliseconds())
	return nil
}

func (c *Client) AwaitConnected(timeout time.Duration) error {
	if c.IsConnected() {
		return nil
	}
	if !c.needConnect.Load() {
		return errors.New("VirtualRouterClient 未启动")
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	deadline := time.Now().Add(timeout)
	for {
		if c.IsConnected() {
			return nil
		}
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return errors.New("等待 VirtualRouterClient 重连超时")
		}
		wait := 100 * time.Millisecond
		if remaining < wait {
			wait = remaining
		}
		select {
		case <-c.stopCh:
			return errors.New("VirtualRouterClient 已关闭")
		case <-time.After(wait):
		}
	}
}

func (c *Client) AwaitSystemClose() {
	for {
		time.Sleep(5 * time.Second)
	}
}

func safeData(msg *core.RouteMessage) string {
	if msg.Data == nil {
		return ""
	}
	return *msg.Data
}

func intToString(v int) string {
	return strconv.Itoa(v)
}

func (c *Client) nextReconnectDelay(attempt int) time.Duration {
	baseMs := c.cfg.ReconnectIntervalMs
	if baseMs <= 0 {
		baseMs = 10000
	}
	base := time.Duration(baseMs) * time.Millisecond
	if attempt <= 1 {
		return base
	}
	delay := base
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= 60*time.Second {
			return 60 * time.Second
		}
	}
	if delay > 60*time.Second {
		return 60 * time.Second
	}
	return delay
}

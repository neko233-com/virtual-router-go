package VirtualRouterClient

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"virtual-router-go/internal/config"
	"virtual-router-go/internal/core"
	"virtual-router-go/internal/rpc"
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

	log.Printf("RPC 模式: %s", strings.ToUpper(cfg.RpcMode))
	return c, nil
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
		log.Printf("RPC 模式: DIRECT - 启动本地 RPC 服务器 port=%d", c.cfg.LocalRpcPort)
		server := rpc.NewStubServer(c.cfg.LocalRpcPort)
		go server.Start()
	} else {
		log.Printf("RPC 模式: RELAY - RPC 调用将通过 Router Center 转发")
	}
}

func (c *Client) runRouterClient() {
	if c.tryConnect() {
		log.Printf("✅ 连接 Router Center 成功! %s:%d, current routeId = %s", c.routerCenterHost, c.routerCenterPort, c.routeId)
		c.isOpen.Store(true)
		c.startHeartbeat()
		go c.readLoop()
		return
	}
	log.Printf("❌ 首次连接 Router Center 失败! %s:%d, 将在后台自动重连...", c.routerCenterHost, c.routerCenterPort)
	c.startBackgroundReconnect()
}

func (c *Client) tryConnect() bool {
	conn, err := net.Dial("tcp", net.JoinHostPort(c.routerCenterHost, intToString(c.routerCenterPort)))
	if err != nil {
		return false
	}
	c.conn = conn
	c.reconnectAttempt.Store(false)
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
				c.isOpen.Store(false)
				log.Printf("Router Center 离线! host=%s port=%d", c.routerCenterHost, c.routerCenterPort)
				c.startBackgroundReconnect()
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
			c.isOpen.Store(false)
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
		log.Printf("收到 data = %s", safeData(msg))
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
		log.Printf("init route info error: %v", err)
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
		log.Printf("remove offline parse error: %v", err)
		return
	}
	if len(ids) == 0 {
		return
	}
	RouteTableInstance().RemoveRouteNode(ids)
	log.Printf("[Client] 删除已离线的 Route Client. toRemoteRouteIdList = %v", ids)
}

func (c *Client) handleSystemError(msg *core.RouteMessage) {
	if msg.Data == nil {
		return
	}
	errMsg := *msg.Data
	log.Printf("❌ 收到系统错误: %s", errMsg)
	if strings.Contains(errMsg, "RouterId") && strings.Contains(errMsg, "已经存在") {
		log.Printf("==================================================")
		log.Printf("   FATAL ERROR: RouterId 冲突")
		log.Printf("   %s", errMsg)
		log.Printf("   请修改配置文件中的 routeId，然后重启程序。")
		log.Printf("==================================================")
		panic(errMsg)
	}
}

func (c *Client) startBackgroundReconnect() {
	if !c.reconnectAttempt.CompareAndSwap(false, true) {
		return
	}
	go func() {
		for {
			if !c.needConnect.Load() {
				return
			}
			if !c.isOpen.Load() {
				if c.tryConnect() {
					c.isOpen.Store(true)
					c.startHeartbeat()
					go c.readLoop()
					return
				}
			}
			select {
			case <-c.stopCh:
				return
			case <-time.After(time.Duration(c.cfg.ReconnectIntervalMs) * time.Millisecond):
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
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_, err = c.conn.Write(core.EncodeFrame(payload))
	return err
}

func (c *Client) IsConnected() bool {
	return c.isOpen.Load() && c.conn != nil
}

func (c *Client) Shutdown() {
	c.needConnect.Store(false)
	c.isOpen.Store(false)
	close(c.stopCh)
	if c.conn != nil {
		_ = c.conn.Close()
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
	log.Printf("等待 router-server 返回 rpc 信息总共等了 %d ms", time.Since(start).Milliseconds())
	return nil
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

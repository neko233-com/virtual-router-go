package rpc

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"log"
	"net"
	"sync"
	"time"
)

type DirectClient struct {
	localRouteId string
	routeId      string
	host         string
	port         int
	conn         net.Conn
	mu           sync.Mutex
}

func NewDirectClient(localRouteId, routeId, host string, port int) *DirectClient {
	return &DirectClient{localRouteId: localRouteId, routeId: routeId, host: host, port: port}
}

func (c *DirectClient) Start() {
	conn, err := net.Dial("tcp", net.JoinHostPort(c.host, intToString(c.port)))
	if err != nil {
		log.Printf("rpc client connect error routeId=%s: %v", c.routeId, err)
		return
	}
	c.conn = conn
	defer c.Close()

	for {
		msg, err := readRpcFrame(conn)
		if err != nil {
			return
		}
		var resp RpcResponse
		if err := json.Unmarshal(msg, &resp); err != nil {
			continue
		}
		f := WaitResultManagerInstance().Pop(resp.RpcUid)
		if f == nil {
			continue
		}
		if resp.ErrorFlag {
			f.Error(resp.ErrorMsg)
		} else {
			f.Success(resp.ResultValueStr)
		}
	}
}

func (c *DirectClient) SendRpcMessage(request *RpcRequest) (bool, error) {
	if c.conn == nil {
		return false, errors.New("rpc client 未连接")
	}
	b, _ := json.Marshal(request)
	frame := make([]byte, 4+len(b))
	binary.BigEndian.PutUint32(frame[:4], uint32(len(b)))
	copy(frame[4:], b)
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.conn.Write(frame)
	return err == nil, err
}

func (c *DirectClient) GetOrCreateProxy(packetId int, timeout time.Duration, args []json.RawMessage) (string, error) {
	req := &RpcRequest{
		FromRouteId:        c.localRouteId,
		ToRouteId:          c.routeId,
		RpcUid:             GenerateRpcUid(),
		StartTimeMs:        time.Now().UnixMilli(),
		PacketId:           packetId,
		MethodArgsJsonList: rawToStringList(args),
	}
	ok, err := c.SendRpcMessage(req)
	if !ok || err != nil {
		return "", err
	}
	future := NewFuture(req.RpcUid)
	WaitResultManagerInstance().Register(future)
	return future.Await(timeout)
}

func (c *DirectClient) Close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

package rpc

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"log"
	"net"
	"sync"
)

type StubServer struct {
	port int
}

func NewStubServer(port int) *StubServer {
	return &StubServer{port: port}
}

func (s *StubServer) Start() error {
	ln, err := net.Listen("tcp", ":"+intToString(s.port))
	if err != nil {
		return err
	}
	log.Printf("RPC Server - 启动成功! port=%d", s.port)
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		go s.handleConn(conn)
	}
}

func (s *StubServer) handleConn(conn net.Conn) {
	defer conn.Close()
	for {
		msg, err := readRpcFrame(conn)
		if err != nil {
			return
		}
		var req RpcRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			continue
		}
		if req.RpcUid == "" {
			continue
		}
		response := RpcResponse{RpcUid: req.RpcUid, StartTimeMs: req.StartTimeMs, PacketId: req.PacketId}

		result, err := ServerStubManagerInstance().Invoke(req.PacketId, rawToJsonArgs(req.MethodArgsJsonList))
		if err != nil {
			response.ErrorFlag = true
			response.ErrorMsg = err.Error()
		} else {
			response.ResultValueStr = toJsonOrString(result)
		}
		respBytes, _ := json.Marshal(response)
		_ = writeRpcFrame(conn, respBytes)
	}
}

func toJsonOrString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func readRpcFrame(conn net.Conn) ([]byte, error) {
	lenBuf := make([]byte, 4)
	if _, err := conn.Read(lenBuf); err != nil {
		return nil, err
	}
	length := int(binary.BigEndian.Uint32(lenBuf))
	if length <= 0 || length > 1024*1024 {
		return nil, ErrFrameTooLarge
	}
	buf := make([]byte, length)
	_, err := io.ReadFull(conn, buf)
	return buf, err
}

func writeRpcFrame(conn net.Conn, payload []byte) error {
	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock()
	buf := make([]byte, 4+len(payload))
	binary.BigEndian.PutUint32(buf[:4], uint32(len(payload)))
	copy(buf[4:], payload)
	_, err := conn.Write(buf)
	return err
}

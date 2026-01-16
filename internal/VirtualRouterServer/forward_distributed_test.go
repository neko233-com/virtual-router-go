package VirtualRouterServer

import (
	"net"
	"sync"
	"testing"

	"virtual-router-go/internal/config"
	"virtual-router-go/internal/core"
)

func TestForwardToTarget_Distributed(t *testing.T) {
	srv := NewServer(&config.RouterServerConfig{RouterServerPort: 1, HTTPMonitorPort: 2})

	connA, connAClient := net.Pipe()
	connB, connBClient := net.Pipe()
	defer connA.Close()
	defer connAClient.Close()
	defer connB.Close()
	defer connBClient.Close()

	sessionA := NewRouterSession("A", connA, core.RpcServerInfo{}, &sync.Mutex{})
	sessionB := NewRouterSession("B", connB, core.RpcServerInfo{}, &sync.Mutex{})

	if _, err := srv.SessionManager().UpsertSession("A", sessionA); err != nil {
		t.Fatalf("upsert session A error: %v", err)
	}
	if _, err := srv.SessionManager().UpsertSession("B", sessionB); err != nil {
		t.Fatalf("upsert session B error: %v", err)
	}

	data := `{"hello":"world"}`
	mt := core.RouteMessageTypeRpcRequest
	msg := &core.RouteMessage{
		FromRouteId: "A",
		ToRouteId:   "B",
		MessageType: &mt,
		Data:        &data,
	}

	readCh := make(chan []byte, 1)
	errCh := make(chan error, 1)

	go func() {
		payload, err := core.ReadFrame(connBClient)
		if err != nil {
			errCh <- err
			return
		}
		readCh <- payload
	}()

	go srv.forwardToTarget(msg)

	select {
	case payload := <-readCh:
		received, err := core.DecodeRouteMessagePayload(payload)
		if err != nil {
			t.Fatalf("decode message error: %v", err)
		}

		if received.FromRouteId != "A" || received.ToRouteId != "B" || received.MessageType == nil || *received.MessageType != core.RouteMessageTypeRpcRequest {
			t.Fatalf("unexpected message: %+v", received)
		}
		if received.Data == nil || *received.Data != data {
			t.Fatalf("unexpected data: %+v", received.Data)
		}
	case err := <-errCh:
		t.Fatalf("read frame error: %v", err)
	}
}

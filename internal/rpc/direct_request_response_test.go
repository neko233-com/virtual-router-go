package rpc

import (
	"encoding/json"
	"errors"
	"net"
	"testing"
	"time"
)

func TestDirectRpcRequestResponse(t *testing.T) {
	ServerStubManagerInstance().Reset()

	if err := RegisterRpcFunc(RpcFuncMeta{PacketId: 10, Description: "mul"}, func(a int, b int) (int, error) {
		return a * b, nil
	}); err != nil {
		t.Fatalf("RegisterRpcFunc error: %v", err)
	}

	port := getFreePort(t)
	server := NewStubServer(port)
	go func() {
		_ = server.Start()
	}()
	time.Sleep(50 * time.Millisecond)

	client := NewDirectClient("clientA", "serverB", "127.0.0.1", port)
	go client.Start()
	defer client.Close()

	if err := waitForConn(client, 2*time.Second); err != nil {
		t.Fatalf("client connect error: %v", err)
	}

	res, err := client.GetOrCreateProxy(10, time.Second, []json.RawMessage{json.RawMessage("4"), json.RawMessage("5")})
	if err != nil {
		t.Fatalf("rpc call error: %v", err)
	}
	if res != "20" {
		t.Fatalf("unexpected result: %s", res)
	}
}

func getFreePort(t *testing.T) int {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func waitForConn(client *DirectClient, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if client.conn != nil {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return errors.New("rpc client connect timeout")
}

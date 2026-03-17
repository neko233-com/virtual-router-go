package rpc_test

import (
	"encoding/json"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/neko233-com/virtual-router-go/internal/rpc"
)

func TestDirectRpcRequestResponse(t *testing.T) {
	rpc.ServerStubManagerInstance().Reset()

	if err := rpc.RegisterRpcFunc(rpc.RpcFuncMeta{PacketId: 10, Description: "mul"}, func(a int, b int) (int, error) {
		return a * b, nil
	}); err != nil {
		t.Fatalf("RegisterRpcFunc error: %v", err)
	}

	port := getFreePort(t)
	server := rpc.NewStubServer(port)
	go func() {
		_ = server.Start()
	}()
	time.Sleep(50 * time.Millisecond)

	client := rpc.NewDirectClient("clientA", "serverB", "127.0.0.1", port)
	go client.Start()
	defer client.Close()

	if err := waitForReady(client, 2*time.Second); err != nil {
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

func waitForReady(client *rpc.DirectClient, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := client.GetOrCreateProxy(10, 100*time.Millisecond, []json.RawMessage{json.RawMessage("1"), json.RawMessage("1")}); err == nil {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return errors.New("rpc client connect timeout")
}

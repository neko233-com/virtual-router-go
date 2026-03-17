package rpc_test

import (
	"encoding/json"
	"testing"

	"github.com/neko233-com/virtual-router-go/internal/core"
	"github.com/neko233-com/virtual-router-go/internal/rpc"
)

func TestStubManagerRegisterInvoke(t *testing.T) {
	mgr := rpc.ServerStubManagerInstance()
	mgr.Reset()
	defer mgr.Reset()

	meta := core.RpcStubMetadata{
		PacketId:       1,
		Description:    "echo",
		ClassName:      "Test",
		MethodName:     "Echo",
		ParameterTypes: []string{"string"},
	}

	mgr.RegisterStub(meta, func(args []json.RawMessage) (any, error) {
		if len(args) != 1 {
			return "", nil
		}
		var s string
		_ = json.Unmarshal(args[0], &s)
		return s, nil
	})

	out, err := mgr.Invoke(1, []json.RawMessage{json.RawMessage([]byte("\"hi\""))})
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if out.(string) != "hi" {
		t.Fatalf("unexpected result: %v", out)
	}
}

func TestStubManagerInvoke_PanicHandlerShouldReturnError(t *testing.T) {
	mgr := rpc.ServerStubManagerInstance()
	mgr.Reset()
	defer mgr.Reset()

	meta := core.RpcStubMetadata{PacketId: 2, Description: "panic", ClassName: "Test", MethodName: "Panic"}
	mgr.RegisterStub(meta, func(args []json.RawMessage) (any, error) {
		panic("boom")
	})

	_, err := mgr.Invoke(2, nil)
	if err == nil {
		t.Fatal("expected panic converted to error, got nil")
	}
}

func TestStubManagerCheckInitialized_ReturnsError(t *testing.T) {
	mgr := rpc.ServerStubManagerInstance()
	mgr.Reset()
	defer mgr.Reset()

	if err := mgr.CheckInitialized(); err == nil {
		t.Fatal("expected initialization error when no stubs registered")
	}
}


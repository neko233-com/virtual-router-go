package rpc

import (
	"encoding/json"
	"testing"

	"virtual-router-go/internal/core"
)

func TestStubManagerRegisterInvoke(t *testing.T) {
	mgr := ServerStubManagerInstance()

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

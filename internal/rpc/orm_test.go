package rpc

import (
	"encoding/json"
	"testing"
)

type testUser struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestRegisterRpcFuncAndInvoke(t *testing.T) {
	ServerStubManagerInstance().Reset()

	meta := RpcFuncMeta{
		PacketId:    1,
		Description: "add",
		ParamMeta: []RpcParamMeta{
			{Name: "a"},
			{Name: "b"},
		},
	}

	err := RegisterRpcFunc(meta, func(a int, b int) (int, error) {
		return a + b, nil
	})
	if err != nil {
		t.Fatalf("RegisterRpcFunc error: %v", err)
	}

	result, err := ServerStubManagerInstance().Invoke(1, []json.RawMessage{json.RawMessage("2"), json.RawMessage("3")})
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}

	if v, ok := result.(int); !ok || v != 5 {
		t.Fatalf("unexpected result: %#v", result)
	}

	stubs := ServerStubManagerInstance().GetAllStubsMetadata()
	if len(stubs) != 1 {
		t.Fatalf("expected 1 stub, got %d", len(stubs))
	}
	if stubs[0].ParameterNames[0] != "a" || stubs[0].ParameterNames[1] != "b" {
		t.Fatalf("unexpected parameter names: %#v", stubs[0].ParameterNames)
	}
}

func TestRegisterRpcFuncWithStructArg(t *testing.T) {
	ServerStubManagerInstance().Reset()

	meta := RpcFuncMeta{PacketId: 2, Description: "echo"}
	if err := RegisterRpcFunc(meta, func(u testUser) (string, error) {
		return u.Name, nil
	}); err != nil {
		t.Fatalf("RegisterRpcFunc error: %v", err)
	}

	payload := json.RawMessage(`{"name":"neo","age":7}`)
	result, err := ServerStubManagerInstance().Invoke(2, []json.RawMessage{payload})
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	if result != "neo" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

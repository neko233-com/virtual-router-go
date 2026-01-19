package VirtualRouterClient

import (
	"encoding/json"

	"github.com/neko233-com/virtual-router-go/internal/core"
	"github.com/neko233-com/virtual-router-go/internal/rpc"
)

// RegisterRpcStub 注册 RPC Stub（供外部服务实现注册）
func RegisterRpcStub(meta RpcStubMetadata, handler func(args []json.RawMessage) (any, error)) {
	rpc.ServerStubManagerInstance().RegisterStub(core.RpcStubMetadata(meta), handler)
}

// RegisterRpcFunc 使用函数签名自动完成参数反序列化和元数据注册
func RegisterRpcFunc(meta RpcFuncMeta, fn any) error {
	return rpc.RegisterRpcFunc(meta, fn)
}

// EnsureStubInitialized 检查是否已注册
func EnsureStubInitialized() {
	rpc.ServerStubManagerInstance().EnsureInitialized()
}

package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/neko233-com/virtual-router-go/internal/core"
)

type RpcHandler func(args []json.RawMessage) (any, error)

type StubManager struct {
	mu             sync.RWMutex
	initialized    atomic.Bool
	handlers       map[int]RpcHandler
	metadata       map[int]core.RpcStubMetadata
	interfaceIndex map[string]any
}

var stubManagerInstance = &StubManager{
	handlers:       map[int]RpcHandler{},
	metadata:       map[int]core.RpcStubMetadata{},
	interfaceIndex: map[string]any{},
}

func ServerStubManagerInstance() *StubManager {
	return stubManagerInstance
}

func (m *StubManager) EnsureInitialized() {
	if err := m.CheckInitialized(); err != nil {
		slog.Error("RPC Stub 尚未初始化", "error", err)
	}
}

func (m *StubManager) CheckInitialized() error {
	if !m.initialized.Load() {
		return errors.New("还没有调用 StubManager.RegisterStub 初始化 rpc server method stub")
	}
	return nil
}

func (m *StubManager) RegisterStub(meta core.RpcStubMetadata, handler RpcHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[meta.PacketId] = handler
	m.metadata[meta.PacketId] = meta
	m.initialized.Store(true)
}

func (m *StubManager) RegisterInterface(name string, instance any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.interfaceIndex[name] = instance
}

func (m *StubManager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = map[int]RpcHandler{}
	m.metadata = map[int]core.RpcStubMetadata{}
	m.interfaceIndex = map[string]any{}
	m.initialized.Store(false)
}

func (m *StubManager) GetHandler(packetId int) (RpcHandler, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	h, ok := m.handlers[packetId]
	return h, ok
}

func (m *StubManager) GetAllStubsMetadata() []core.RpcStubMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make([]core.RpcStubMetadata, 0, len(m.metadata))
	for _, v := range m.metadata {
		list = append(list, v)
	}
	return list
}

func (m *StubManager) Invoke(packetId int, args []json.RawMessage) (result any, err error) {
	h, ok := m.GetHandler(packetId)
	if !ok {
		return nil, errors.New("方法未注册: packetId=" + intToString(packetId))
	}
	defer func() {
		if r := recover(); r != nil {
			slog.Error("RPC 执行发生 panic，已恢复避免进程崩溃", "packetId", packetId, "panic", r)
			err = fmt.Errorf("RPC 执行异常: packetId=%d panic=%v", packetId, r)
		}
	}()
	result, err = h(args)
	return result, err
}

func intToString(v int) string {
	return strconv.Itoa(v)
}

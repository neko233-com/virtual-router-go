package rpc

import (
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"

	"strconv"
	"virtual-router-go/internal/core"
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
	if !m.initialized.Load() {
		panic("还没有调用 StubManager.RegisterStub 初始化 rpc server method stub")
	}
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

func (m *StubManager) Invoke(packetId int, args []json.RawMessage) (any, error) {
	h, ok := m.GetHandler(packetId)
	if !ok {
		return nil, errors.New("方法未注册: packetId=" + intToString(packetId))
	}
	return h(args)
}

func intToString(v int) string {
	return strconv.Itoa(v)
}

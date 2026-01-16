package rpc

import (
	"errors"
	"sync"
	"time"
)

type Future struct {
	rpcUid string
	ch     chan struct{}
	result string
	err    error
}

func NewFuture(rpcUid string) *Future {
	return &Future{
		rpcUid: rpcUid,
		ch:     make(chan struct{}),
	}
}

func (f *Future) Success(result string) {
	f.result = result
	close(f.ch)
}

func (f *Future) Error(msg string) {
	f.err = errors.New(msg)
	close(f.ch)
}

func (f *Future) Await(timeout time.Duration) (string, error) {
	if timeout <= 0 {
		<-f.ch
		return f.result, f.err
	}
	select {
	case <-f.ch:
		return f.result, f.err
	case <-time.After(timeout):
		return "", errors.New("rpc timeout")
	}
}

// FutureManager 用于管理等待中的 RPC Future

type FutureManager struct {
	mu sync.Mutex
	m  map[string]*Future
}

func NewFutureManager() *FutureManager {
	return &FutureManager{m: map[string]*Future{}}
}

func (fm *FutureManager) Register(f *Future) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.m[f.rpcUid] = f
}

func (fm *FutureManager) Pop(rpcUid string) *Future {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	f := fm.m[rpcUid]
	delete(fm.m, rpcUid)
	return f
}

func (fm *FutureManager) SetSuccess(rpcUid, result string) {
	if f := fm.Pop(rpcUid); f != nil {
		f.Success(result)
	}
}

func (fm *FutureManager) SetError(rpcUid, errMsg string) {
	if f := fm.Pop(rpcUid); f != nil {
		f.Error(errMsg)
	}
}

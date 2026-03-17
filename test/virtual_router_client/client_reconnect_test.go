package virtual_router_client_test

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	clientpkg "github.com/neko233-com/virtual-router-go/internal/VirtualRouterClient"
	"github.com/neko233-com/virtual-router-go/internal/config"
	"github.com/neko233-com/virtual-router-go/internal/core"
	"github.com/neko233-com/virtual-router-go/internal/rpc"
)

// waitUntil 在超时时间内轮询条件函数，常用于异步重连场景断言。
func waitUntil(timeout time.Duration, check func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

func TestClient_冲突后不崩溃且可自动重连(t *testing.T) {
	// 1) 注册最小 RPC Stub，避免 Start() 时 EnsureInitialized 触发 panic。
	rpc.ServerStubManagerInstance().Reset()
	defer rpc.ServerStubManagerInstance().Reset()
	if err := rpc.RegisterRpcFunc(rpc.RpcFuncMeta{PacketId: 900001, Description: "ut-ping"}, func() (string, error) {
		return "pong", nil
	}); err != nil {
		t.Fatalf("注册测试 RPC Stub 失败: %v", err)
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("启动测试 Router Center 监听失败: %v", err)
	}
	defer func() { _ = ln.Close() }()

	port := ln.Addr().(*net.TCPAddr).Port
	firstConflictDone := make(chan struct{})
	secondConnected := make(chan struct{})
	serverErrCh := make(chan error, 1)

	// 2) 模拟 Router Center：第一次连接返回 RouterId 冲突并断开；第二次连接接受心跳。
	go func() {
		defer close(firstConflictDone)

		// 第一次连接：读取心跳后返回系统错误（RouterId 冲突）。
		conn1, acceptErr := ln.Accept()
		if acceptErr != nil {
			serverErrCh <- fmt.Errorf("首次 accept 失败: %w", acceptErr)
			return
		}
		if _, readErr := core.ReadFrame(conn1); readErr != nil {
			_ = conn1.Close()
			serverErrCh <- fmt.Errorf("首次读取心跳失败: %w", readErr)
			return
		}

		errMsg := "RouterId 'game-server-1' 已经存在! 请修改您的 routerId 配置."
		mt := core.RouteMessageTypeSystemError
		sysErrMsg := &core.RouteMessage{FromRouteId: "server", ToRouteId: "game-server-1", MessageType: &mt, Data: &errMsg}
		payload, encErr := sysErrMsg.EncodePayload()
		if encErr != nil {
			_ = conn1.Close()
			serverErrCh <- fmt.Errorf("首次编码系统错误失败: %w", encErr)
			return
		}
		if _, writeErr := conn1.Write(core.EncodeFrame(payload)); writeErr != nil {
			_ = conn1.Close()
			serverErrCh <- fmt.Errorf("首次写入系统错误失败: %w", writeErr)
			return
		}
		_ = conn1.Close()

		// 第二次连接：验证客户端仍会重连。
		conn2, acceptErr := ln.Accept()
		if acceptErr != nil {
			serverErrCh <- fmt.Errorf("二次 accept 失败: %w", acceptErr)
			return
		}
		defer func() { _ = conn2.Close() }()

		if _, readErr := core.ReadFrame(conn2); readErr != nil {
			serverErrCh <- fmt.Errorf("二次读取心跳失败: %w", readErr)
			return
		}
		close(secondConnected)

		// 给客户端一点稳定窗口，避免测试过快结束导致偶发竞态。
		time.Sleep(100 * time.Millisecond)
	}()

	cfg := &config.RouterClientConfig{
		RouteId:                 "game-server-1",
		RouterCenterHost:        "127.0.0.1",
		RouterCenterPort:        port,
		RpcMode:                 "relay",
		HeartBeatIntervalSecond: 1,
		ReconnectIntervalMs:     50,
	}
	client := clientpkg.NewClientByConfig(cfg)
	if client == nil {
		t.Fatal("创建客户端失败: NewClientByConfig 返回 nil")
	}
	defer client.Shutdown()

	// 3) 启动前 AwaitConnected 应明确提示“未启动”。
	if err := client.AwaitConnected(100 * time.Millisecond); err == nil || !strings.Contains(err.Error(), "未启动") {
		t.Fatalf("启动前 AwaitConnected 返回不符合预期, err=%v", err)
	}

	if err := client.Start(); err != nil {
		t.Fatalf("启动客户端失败: %v", err)
	}

	// 4) 等待服务端完成“冲突下发”阶段。
	select {
	case <-firstConflictDone:
	case err := <-serverErrCh:
		t.Fatalf("测试服务端异常: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("等待冲突阶段超时")
	}

	// 5) 核心断言：客户端不会崩溃，并且能自动发起第二次连接。
	select {
	case <-secondConnected:
	case err := <-serverErrCh:
		t.Fatalf("测试服务端异常: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("冲突后客户端未发生自动重连")
	}

	if ok := waitUntil(2*time.Second, client.IsConnected); !ok {
		t.Fatal("重连后客户端未进入已连接状态")
	}
}

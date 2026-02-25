package main

import (
	"log/slog"
	"os"

	"github.com/neko233-com/virtual-router-go/VirtualRouterClient"
)

func main() {
	client, err := VirtualRouterClient.NewClient("")
	if err != nil {
		slog.Error("创建客户端失败", "error", err)
		os.Exit(1)
	}
	if err := client.Start(); err != nil {
		slog.Error("启动客户端失败", "error", err)
		os.Exit(1)
	}

	if err := client.AwaitRpcRouterInfoFirstReady(); err != nil {
		slog.Error("等待路由信息失败", "error", err)
		os.Exit(1)
	}

	client.AwaitSystemClose()
}

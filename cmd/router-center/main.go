package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/neko233-com/virtual-router-go/VirtualRouterServer"
)

func main() {
	VirtualRouterServer.InstallProcessLogCapture(800)

	cfg, err := VirtualRouterServer.ReadRouterServerConfig("")
	if err != nil {
		slog.Error("读取服务端配置失败", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := VirtualRouterServer.NewServer(cfg)
	httpSrv := VirtualRouterServer.NewHttpServer(cfg, srv)

	go func() {
		if err := srv.Start(ctx); err != nil {
			slog.Error("router server start error", "error", err)
			os.Exit(1)
		}
	}()

	go func() {
		if err := httpSrv.Start(ctx); err != nil {
			slog.Warn("http server stop", "error", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	cancel()
}

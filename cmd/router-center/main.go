package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"virtual-router-go/VirtualRouterServer"
)

func main() {
	cfg, err := VirtualRouterServer.ReadRouterServerConfig("")
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := VirtualRouterServer.NewServer(cfg)
	httpSrv := VirtualRouterServer.NewHttpServer(cfg, srv)

	go func() {
		if err := srv.Start(ctx); err != nil {
			log.Fatalf("router server start error: %v", err)
		}
	}()

	go func() {
		if err := httpSrv.Start(ctx); err != nil {
			log.Printf("http server stop: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	cancel()
}

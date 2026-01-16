package VirtualRouterServer

import (
	"context"

	server "virtual-router-go/internal/VirtualRouterServer"
	"virtual-router-go/internal/config"
)

type Server = server.Server

type HttpServer = server.HttpServer

type RouterServerConfig = config.RouterServerConfig

func NewServer(cfg *config.RouterServerConfig) *server.Server {
	return server.NewServer(cfg)
}

func NewHttpServer(cfg *config.RouterServerConfig, srv *server.Server) *server.HttpServer {
	return server.NewHttpServer(cfg, srv)
}

func ReadRouterServerConfig(fileName string) (*config.RouterServerConfig, error) {
	return config.ReadRouterServerConfig(fileName)
}

func StartServer(ctx context.Context, cfg *config.RouterServerConfig) (*server.Server, *server.HttpServer, error) {
	srv := server.NewServer(cfg)
	httpSrv := server.NewHttpServer(cfg, srv)
	go func() {
		_ = srv.Start(ctx)
	}()
	go func() {
		_ = httpSrv.Start(ctx)
	}()
	return srv, httpSrv, nil
}

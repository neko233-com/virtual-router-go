package config

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
)

const (
	RouterServerConfigName = "neko233-router-server.json"
	RouterClientConfigName = "neko233-router-client.json"
)

// RouterServerConfig 路由服务器配置
type RouterServerConfig struct {
	// 路由服务器监听端口，用于接收来自 Client 的 TCP 连接
	RouterServerPort int `json:"routerServerPort"`
	// HTTP 监控/管理端口，提供后台界面或 API
	HTTPMonitorPort int `json:"httpMonitorPort"`
	// 管理员密码，用于登录监控后台
	AdminPassword string `json:"adminPassword"`
}

// RouterClientConfig 路由客户端配置
type RouterClientConfig struct {
	// 路由节点唯一 ID (必填)，用于标识该客户端
	RouteId string `json:"routeId"`
	// Router Center 的 IP 地址或域名
	RouterCenterHost string `json:"routerCenterHost"`
	// Router Center 的连接端口
	RouterCenterPort int `json:"routerCenterPort"`
	// RPC 模式：'relay' (转发模式) 或 'direct' (直连模式)
	RpcMode string `json:"rpcMode"`
	// 直连模式下，本机的 RPC 服务公网/可访问 IP
	LocalRpcHost string `json:"localRpcHost"`
	// 直连模式下，本机的 RPC 服务监听端口
	LocalRpcPort int `json:"localRpcPort"`
	// 心跳间隔时长（秒）
	HeartBeatIntervalSecond int64 `json:"heartBeatIntervalSecond"`
	// 断线重连尝试间隔（毫秒）
	ReconnectIntervalMs int64 `json:"reconnectIntervalMs"`
}

func ReadRouterServerConfig(fileName string) (*RouterServerConfig, error) {
	if fileName == "" {
		fileName = RouterServerConfigName
	}
	if _, err := os.Stat(fileName); errors.Is(err, os.ErrNotExist) {
		cfg := &RouterServerConfig{RouterServerPort: 9999, HTTPMonitorPort: 19999, AdminPassword: "root"}
		_ = writeDefault(fileName, cfg)
		return nil, errors.New("没有在当前路径找到配置文件, 自动给你生成了一个 " + fileName + ", 配置好后再启动项目!")
	}
	data, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	cfg := &RouterServerConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if cfg.RouterServerPort == 0 {
		return nil, errors.New("请再检查一下 " + fileName + " 的配置, 不允许 routerServerPort = 0")
	}
	if cfg.HTTPMonitorPort == 0 {
		cfg.HTTPMonitorPort = 19999
	}
	if cfg.AdminPassword == "" {
		cfg.AdminPassword = "root"
	}
	return cfg, nil
}

func NewDefaultRouterClientConfig() *RouterClientConfig {
	return &RouterClientConfig{
		RpcMode:                 "relay",
		HeartBeatIntervalSecond: 10,
		ReconnectIntervalMs:     30000,
	}
}

func (cfg *RouterClientConfig) Check() error {
	if strings.TrimSpace(cfg.RouteId) == "" {
		return errors.New("配置错误: 不允许 routeId 为空")
	}
	if strings.TrimSpace(cfg.RouterCenterHost) == "" {
		return errors.New("配置错误: 不允许 routerCenterHost 为空")
	}
	if cfg.RouterCenterPort == 0 {
		return errors.New("配置错误: 不允许 routerCenterPort = 0")
	}
	if cfg.RpcMode == "" {
		cfg.RpcMode = "relay"
	}
	if cfg.HeartBeatIntervalSecond <= 0 {
		cfg.HeartBeatIntervalSecond = 10
	}
	if cfg.ReconnectIntervalMs <= 0 {
		cfg.ReconnectIntervalMs = 30000
	}
	isDirect := strings.EqualFold(cfg.RpcMode, "direct")
	if isDirect {
		if strings.TrimSpace(cfg.LocalRpcHost) == "" {
			return errors.New("direct 模式下，必须配置 localRpcHost")
		}
		if cfg.LocalRpcPort == 0 {
			return errors.New("direct 模式下，必须配置 localRpcPort")
		}
	}
	return nil
}

func ReadRouterClientConfig(fileName string) (*RouterClientConfig, error) {
	if fileName == "" {
		fileName = RouterClientConfigName
	}
	if _, err := os.Stat(fileName); errors.Is(err, os.ErrNotExist) {
		cfg := NewDefaultRouterClientConfig()
		_ = writeDefault(fileName, cfg)
		return nil, errors.New("没有在当前路径找到配置文件, 自动给你生成了一个 " + fileName + ", 配置好后再启动项目!")
	}
	data, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}
	cfg := &RouterClientConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	if err := cfg.Check(); err != nil {
		return nil, errors.New("请再检查一下 " + fileName + " 的配置, " + err.Error())
	}
	return cfg, nil
}

func writeDefault(fileName string, cfg any) error {
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(fileName, b, 0644)
}

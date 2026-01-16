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

type RouterServerConfig struct {
	RouterServerPort int    `json:"routerServerPort"`
	HTTPMonitorPort  int    `json:"httpMonitorPort"`
	AdminPassword    string `json:"adminPassword"`
}

type RouterClientConfig struct {
	RouteId                 string `json:"routeId"`
	RouterCenterHost        string `json:"routerCenterHost"`
	RouterCenterPort        int    `json:"routerCenterPort"`
	RpcMode                 string `json:"rpcMode"`
	LocalRpcHost            string `json:"localRpcHost"`
	LocalRpcPort            int    `json:"localRpcPort"`
	HeartBeatIntervalSecond int64  `json:"heartBeatIntervalSecond"`
	ReconnectIntervalMs     int64  `json:"reconnectIntervalMs"`
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

func ReadRouterClientConfig(fileName string) (*RouterClientConfig, error) {
	if fileName == "" {
		fileName = RouterClientConfigName
	}
	if _, err := os.Stat(fileName); errors.Is(err, os.ErrNotExist) {
		cfg := &RouterClientConfig{
			RpcMode:                 "relay",
			HeartBeatIntervalSecond: 10,
			ReconnectIntervalMs:     30000,
		}
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
	if strings.TrimSpace(cfg.RouteId) == "" {
		return nil, errors.New("请再检查一下 " + fileName + " 的配置, 不允许 routeId 为空")
	}
	if strings.TrimSpace(cfg.RouterCenterHost) == "" {
		return nil, errors.New("请再检查一下 " + fileName + " 的配置, 不允许 routerCenterHost 为空")
	}
	if cfg.RouterCenterPort == 0 {
		return nil, errors.New("请再检查一下 " + fileName + " 的配置, 不允许 routerCenterPort = 0")
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
			return nil, errors.New("direct 模式下，必须配置 localRpcHost")
		}
		if cfg.LocalRpcPort == 0 {
			return nil, errors.New("direct 模式下，必须配置 localRpcPort")
		}
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

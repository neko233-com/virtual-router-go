package VirtualRouterClient

import "virtual-router-go/internal/config"

type RouterClientConfig = config.RouterClientConfig

func ReadRouterClientConfig(fileName string) (*config.RouterClientConfig, error) {
	return config.ReadRouterClientConfig(fileName)
}

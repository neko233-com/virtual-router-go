package VirtualRouterClient

import "github.com/neko233-com/virtual-router-go/internal/config"

type RouterClientConfig = config.RouterClientConfig

func ReadRouterClientConfig(fileName string) (*config.RouterClientConfig, error) {
	return config.ReadRouterClientConfig(fileName)
}

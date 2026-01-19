package VirtualRouterClient

import (
	internalClient "github.com/neko233-com/virtual-router-go/internal/VirtualRouterClient"
)

type Client struct {
	inner *internalClient.Client
}

func NewClient(configFile string) (*Client, error) {
	c, err := internalClient.NewClient(configFile)
	if err != nil {
		return nil, err
	}
	return &Client{inner: c}, nil
}

func (c *Client) Start() error {
	return c.inner.Start()
}

func (c *Client) Shutdown() {
	c.inner.Shutdown()
}

func (c *Client) IsConnected() bool {
	return c.inner.IsConnected()
}

func (c *Client) RouteId() string {
	return c.inner.RouteId()
}

func (c *Client) Send(toRouteId string, msgType RouteMessageType, obj any) error {
	return c.inner.Send(toRouteId, msgType, obj)
}

func (c *Client) AwaitRpcRouterInfoFirstReady() error {
	return c.inner.AwaitRpcRouterInfoFirstReady()
}

func (c *Client) AwaitSystemClose() {
	c.inner.AwaitSystemClose()
}

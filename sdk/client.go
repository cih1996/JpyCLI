package sdk

import (
	"errors"
	"time"

	wsclient "jpy-cli/pkg/client/ws"
	deviceapi "jpy-cli/pkg/middleware/device/api"
)

// Client is the main entry point for the JPY SDK.
type Client struct {
	BaseURL  string
	Token    string
	WSClient *wsclient.Client
	Device   *deviceapi.DeviceAPI
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
	}
}

func (c *Client) Connect() error {
	if c.BaseURL == "" {
		return errors.New("BaseURL is required")
	}

	c.WSClient = wsclient.NewClient(c.BaseURL, c.Token)
	c.WSClient.Timeout = 10 * time.Second

	if err := c.WSClient.Connect(); err != nil {
		return err
	}

	c.Device = deviceapi.NewDeviceAPI(c.WSClient, c.BaseURL, c.Token)
	return nil
}

func (c *Client) Close() {
	if c.WSClient != nil {
		c.WSClient.Close()
	}
}

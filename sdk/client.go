package sdk

import (
	"errors"
	"time"

	adminapi "jpy-cli/pkg/admin-middleware/api"
	wsclient "jpy-cli/pkg/client/ws"
	deviceapi "jpy-cli/pkg/middleware/device/api"
)

// Client is the main entry point for the JPY SDK.
// It aggregates various API clients (Device, Admin, etc.) into a single interface.
type Client struct {
	// Configuration
	BaseURL string
	Token   string

	// Underlying WebSocket Client
	WSClient *wsclient.Client

	// API Groups
	Device *deviceapi.DeviceAPI
	Admin  *adminapi.Client
}

// NewClient creates a new SDK client instance.
// baseURL: The URL of the middleware server (e.g., "http://localhost:8080" or "ws://localhost:8080").
// token: The authentication token.
func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		// Admin client is initialized immediately as it's HTTP-based
		Admin: adminapi.NewClient(token),
	}
}

// Connect establishes the WebSocket connection required for real-time Device operations.
// This must be called before using the Device API.
func (c *Client) Connect() error {
	if c.BaseURL == "" {
		return errors.New("BaseURL is required")
	}

	// Initialize WebSocket Client
	c.WSClient = wsclient.NewClient(c.BaseURL, c.Token)
	
	// Set a reasonable default timeout
	c.WSClient.Timeout = 10 * time.Second

	if err := c.WSClient.Connect(); err != nil {
		return err
	}

	// Initialize Device API with the connected transport
	// Note: wsclient.Client implicitly satisfies protocol.Transport interface
	c.Device = deviceapi.NewDeviceAPI(c.WSClient, c.BaseURL, c.Token)

	return nil
}

// Close closes the underlying WebSocket connection.
func (c *Client) Close() {
	if c.WSClient != nil {
		c.WSClient.Close()
	}
}

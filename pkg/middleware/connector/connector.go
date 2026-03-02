package connector

import (
	"fmt"
	httpclient "jpy-cli/pkg/client/http"
	wsclient "jpy-cli/pkg/client/ws"
	"jpy-cli/pkg/config"
	"jpy-cli/pkg/logger"
	"strings"
	"time"
)

// ConnectorService handles server connections with auto-login capabilities
type ConnectorService struct {
	Config *config.Config
}

func NewConnectorService(cfg *config.Config) *ConnectorService {
	return &ConnectorService{Config: cfg}
}

// Connect attempts to connect to a WebSocket server.
// If the connection fails due to auth errors (401/403), it attempts to re-login and retry once.
func (s *ConnectorService) Connect(server config.LocalServerConfig) (*wsclient.Client, error) {
	ws := wsclient.NewClient(server.URL, server.Token)

	// Apply global timeout setting if available
	if config.GlobalSettings.ConnectTimeout > 0 {
		ws.Timeout = time.Duration(config.GlobalSettings.ConnectTimeout) * time.Second
	} else {
		ws.Timeout = 3 * time.Second // Default fallback
	}

	err := ws.Connect()
	if err == nil {
		return ws, nil
	}

	// Check for Auth failure
	if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
		logger.Infof("[%s] 认证失败，正在重新登录...", server.URL)

		// Attempt Login
		hc := httpclient.NewClient(server.URL, "")
		token, loginErr := hc.Login(server.Username, server.Password)
		if loginErr != nil {
			ws.Close()
			return nil, fmt.Errorf("认证失败且重新登录失败: %v", loginErr)
		}

		// Update Token in Config
		server.Token = token
		server.LastLoginTime = time.Now().Format(time.RFC3339)
		if err := config.UpdateServer(s.Config, server); err != nil {
			logger.Warnf("持久化新 token 失败 %s: %v", server.URL, err)
		}

		// Retry Connection with new token
		ws.Token = token
		if errRetry := ws.Connect(); errRetry != nil {
			ws.Close()
			return nil, fmt.Errorf("重新登录成功但连接失败: %v", errRetry)
		}

		return ws, nil
	}

	ws.Close()
	return nil, err
}

// ConnectDeviceTerminal connects to the server's guard channel for a specific device (Terminal mode)
func (s *ConnectorService) ConnectDeviceTerminal(server config.LocalServerConfig, deviceID int64) (*wsclient.Client, error) {
	ws := wsclient.NewClient(server.URL, server.Token)
	ws.Endpoint = "/box/guard"
	ws.Params = map[string]string{"id": fmt.Sprintf("%d", deviceID)}

	// Apply global timeout setting if available
	if config.GlobalSettings.ConnectTimeout > 0 {
		ws.Timeout = time.Duration(config.GlobalSettings.ConnectTimeout) * time.Second
	} else {
		ws.Timeout = 5 * time.Second
	}

	err := ws.Connect()
	if err == nil {
		return ws, nil
	}

	// Check for Auth failure and retry (same logic as Connect)
	if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
		logger.Infof("[%s] 认证失败，正在重新登录...", server.URL)

		hc := httpclient.NewClient(server.URL, "")
		token, loginErr := hc.Login(server.Username, server.Password)
		if loginErr != nil {
			ws.Close()
			return nil, fmt.Errorf("认证失败且重新登录失败: %v", loginErr)
		}

		server.Token = token
		server.LastLoginTime = time.Now().Format(time.RFC3339)
		if err := config.UpdateServer(s.Config, server); err != nil {
			logger.Warnf("持久化新 token 失败 %s: %v", server.URL, err)
		}

		ws.Token = token
		if errRetry := ws.Connect(); errRetry != nil {
			ws.Close()
			return nil, fmt.Errorf("重新登录成功但连接失败: %v", errRetry)
		}
		return ws, nil
	}

	ws.Close()
	return nil, err
}

// ConnectGuard connects to the server's guard channel
func (s *ConnectorService) ConnectGuard(server config.LocalServerConfig) (*wsclient.Client, error) {
	ws := wsclient.NewClient(server.URL, server.Token)
	ws.Endpoint = "/box/guard"
	ws.Params = map[string]string{"id": "0"}

	// Apply global timeout setting if available
	if config.GlobalSettings.ConnectTimeout > 0 {
		ws.Timeout = time.Duration(config.GlobalSettings.ConnectTimeout) * time.Second
	} else {
		ws.Timeout = 3 * time.Second // Default fallback
	}

	err := ws.Connect()
	if err == nil {
		return ws, nil
	}

	// Check for Auth failure
	if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
		logger.Infof("[%s] 认证失败，正在重新登录...", server.URL)

		// Attempt Login
		hc := httpclient.NewClient(server.URL, "")
		token, loginErr := hc.Login(server.Username, server.Password)
		if loginErr != nil {
			ws.Close()
			return nil, fmt.Errorf("认证失败且重新登录失败: %v", loginErr)
		}

		// Update Token in Config
		server.Token = token
		server.LastLoginTime = time.Now().Format(time.RFC3339)
		if err := config.UpdateServer(s.Config, server); err != nil {
			logger.Warnf("持久化新 token 失败 %s: %v", server.URL, err)
		}

		// Retry Connection with new token
		ws.Token = token
		if errRetry := ws.Connect(); errRetry != nil {
			ws.Close()
			return nil, fmt.Errorf("重新登录成功但连接失败: %v", errRetry)
		}

		return ws, nil
	}

	ws.Close()
	return nil, err
}

// ConnectMirror connects to the device mirror channel
func (s *ConnectorService) ConnectMirror(server config.LocalServerConfig, seat int) (*wsclient.Client, error) {
	ws := wsclient.NewClient(server.URL, server.Token)
	ws.Endpoint = "/box/mirror"
	ws.Params = map[string]string{"id": fmt.Sprintf("%d", seat)}

	// Apply global timeout setting if available
	if config.GlobalSettings.ConnectTimeout > 0 {
		ws.Timeout = time.Duration(config.GlobalSettings.ConnectTimeout) * time.Second
	} else {
		ws.Timeout = 3 * time.Second // Default fallback
	}

	err := ws.Connect()
	if err == nil {
		return ws, nil
	}

	// Check for Auth failure
	if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
		logger.Infof("[%s] 认证失败，正在重新登录...", server.URL)

		// Attempt Login
		hc := httpclient.NewClient(server.URL, "")
		token, loginErr := hc.Login(server.Username, server.Password)
		if loginErr != nil {
			ws.Close()
			return nil, fmt.Errorf("认证失败且重新登录失败: %v", loginErr)
		}

		// Update Token in Config
		server.Token = token
		server.LastLoginTime = time.Now().Format(time.RFC3339)
		if err := config.UpdateServer(s.Config, server); err != nil {
			logger.Warnf("持久化新 token 失败 %s: %v", server.URL, err)
		}

		// Retry Connection with new token
		ws.Token = token
		if errRetry := ws.Connect(); errRetry != nil {
			ws.Close()
			return nil, fmt.Errorf("重新登录成功但连接失败: %v", errRetry)
		}

		return ws, nil
	}

	ws.Close()
	return nil, err
}

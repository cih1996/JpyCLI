package connector

import (
	"fmt"
	httpclient "jpy-cli/pkg/client/http"
	wsclient "jpy-cli/pkg/client/ws"
	"jpy-cli/pkg/logger"
	"strings"
	"time"
)

const defaultTimeout = 5 * time.Second

// ServerInfo 无状态服务器信息，替代原 config.LocalServerConfig
type ServerInfo struct {
	URL      string
	Username string
	Password string
	Token    string
}

// connectWithRetry 通用连接逻辑：连接 → 401/403 自动重登 → 重试
func connectWithRetry(ws *wsclient.Client, server ServerInfo) (*wsclient.Client, error) {
	err := ws.Connect()
	if err == nil {
		return ws, nil
	}

	// 401/403 自动重新登录
	if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
		logger.Infof("[%s] 认证失败，正在重新登录...", server.URL)

		hc := httpclient.NewClient(server.URL, "")
		token, loginErr := hc.Login(server.Username, server.Password)
		if loginErr != nil {
			ws.Close()
			return nil, fmt.Errorf("认证失败且重新登录失败: %v", loginErr)
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

// Connect 连接到 WebSocket 主通道
func Connect(server ServerInfo) (*wsclient.Client, error) {
	ws := wsclient.NewClient(server.URL, server.Token)
	ws.Timeout = defaultTimeout
	return connectWithRetry(ws, server)
}

// ConnectGuard 连接到 Guard 通道（id=0）
func ConnectGuard(server ServerInfo) (*wsclient.Client, error) {
	ws := wsclient.NewClient(server.URL, server.Token)
	ws.Endpoint = "/box/guard"
	ws.Params = map[string]string{"id": "0"}
	ws.Timeout = defaultTimeout
	return connectWithRetry(ws, server)
}

// ConnectDeviceTerminal 连接到设备 Terminal 通道（id=deviceID）
func ConnectDeviceTerminal(server ServerInfo, deviceID int64) (*wsclient.Client, error) {
	ws := wsclient.NewClient(server.URL, server.Token)
	ws.Endpoint = "/box/guard"
	ws.Params = map[string]string{"id": fmt.Sprintf("%d", deviceID)}
	ws.Timeout = defaultTimeout
	return connectWithRetry(ws, server)
}

// ConnectMirror 连接到设备 Mirror 通道
func ConnectMirror(server ServerInfo, seat int) (*wsclient.Client, error) {
	ws := wsclient.NewClient(server.URL, server.Token)
	ws.Endpoint = "/box/mirror"
	ws.Params = map[string]string{"id": fmt.Sprintf("%d", seat)}
	ws.Timeout = defaultTimeout
	return connectWithRetry(ws, server)
}

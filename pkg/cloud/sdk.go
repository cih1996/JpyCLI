package cloud

import (
	"crypto/tls"
	"fmt"

	"adminApi"
	"adminApi/loginCtl"

	"cnb.cool/accbot/goTool/sessionPkg"
	"cnb.cool/accbot/goTool/wsPkg"
	"github.com/gorilla/websocket"
)

var (
	// Client 全局 SDK 客户端
	Client *adminApi.AdminApi
	// Session 会话
	Session *sessionPkg.Session
	// 内部状态
	serverURL string
)

// InitSDK 初始化 SDK 连接
func InitSDK(url string) error {
	serverURL = url

	// WebSocket 连接
	dialer := *websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	c, _, err := dialer.Dial(serverURL, nil)
	if err != nil {
		return fmt.Errorf("连接服务器失败: %w", err)
	}

	conn := wsPkg.NewWSConnByConn(c)
	newSession := sessionPkg.CreateSession(sessionPkg.SessionType_ws, conn)

	newClient := adminApi.NewAdminApi(newSession)

	Session = newSession
	Client = newClient

	return nil
}

// LoginWithSecret 使用密钥登录
func LoginWithSecret(secretKey string) (*loginCtl.SecretKeyLoginRes, error) {
	if Client == nil {
		return nil, fmt.Errorf("SDK 未初始化")
	}

	req := &loginCtl.SecretKeyLoginReq{
		SecretKey: &secretKey,
	}

	res, errPkg := Client.LoginCtl.SecretKeyLogin(req)
	if errPkg != nil {
		return nil, fmt.Errorf("登录失败: %v", errPkg)
	}
	return res, nil
}

// EnsureConnected 确保已连接并登录
func EnsureConnected() error {
	cfg, err := LoadCloudConfig()
	if err != nil {
		return err
	}

	if cfg.Auth.SecretKey == "" {
		return fmt.Errorf("未配置密钥。请先运行: jpy cloud config --secret-key <key>")
	}

	if Client == nil {
		if err := InitSDK(cfg.ServerURL); err != nil {
			return err
		}

		if _, err := LoginWithSecret(cfg.Auth.SecretKey); err != nil {
			return err
		}
	}

	return nil
}

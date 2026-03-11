package auth

import (
	"fmt"
	httpclient "jpy-cli/pkg/client/http"
	"strings"
)

// ServerCredentials 无状态凭证，每次命令行传入，用完即弃
type ServerCredentials struct {
	ServerURL string
	Username  string
	Password  string
	Token     string
}

// Resolve 从命令行参数解析凭证并自动登录获取 token
// serverURL: 中间件服务器地址（必填）
// username: 用户名（必填）
// password: 密码（必填）
func Resolve(serverURL, username, password string) (*ServerCredentials, error) {
	if serverURL == "" {
		return nil, fmt.Errorf("必须指定 --server / -s 参数")
	}
	if username == "" {
		return nil, fmt.Errorf("必须指定 --username / -u 参数")
	}
	if password == "" {
		return nil, fmt.Errorf("必须指定 --password / -p 参数")
	}

	// 自动补全协议前缀
	if !strings.HasPrefix(serverURL, "http") {
		serverURL = "https://" + serverURL
	}

	// 自动登录获取 token
	client := httpclient.NewClient(serverURL, "")
	token, err := client.Login(username, password)
	if err != nil {
		return nil, fmt.Errorf("登录 %s 失败: %v", serverURL, err)
	}

	return &ServerCredentials{
		ServerURL: serverURL,
		Username:  username,
		Password:  password,
		Token:     token,
	}, nil
}

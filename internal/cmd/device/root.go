package device

import (
	"jpy-cli/pkg/auth"
	"jpy-cli/pkg/middleware/connector"

	"github.com/spf13/cobra"
)

// 公共 flags，所有 device 子命令共享
var (
	flagServer   string
	flagUsername string
	flagPassword string
	flagOutput   string
)

func NewDeviceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "device",
		Short: "设备管理命令",
	}

	// 公共 flags 挂在 device 父命令上，子命令自动继承
	cmd.PersistentFlags().StringVarP(&flagServer, "server", "s", "", "中间件服务器地址（必填）")
	cmd.PersistentFlags().StringVarP(&flagUsername, "username", "u", "", "用户名（必填）")
	cmd.PersistentFlags().StringVarP(&flagPassword, "password", "p", "", "密码（必填）")
	cmd.PersistentFlags().StringVarP(&flagOutput, "output", "o", "plain", "输出模式: plain/json")

	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewShellCmd())
	cmd.AddCommand(NewRebootCmd())
	cmd.AddCommand(NewUSBCmd())
	cmd.AddCommand(NewADBCmd())
	cmd.AddCommand(NewStatusCmd())

	return cmd
}

// resolveCredentials 统一凭证解析
func resolveCredentials() (*auth.ServerCredentials, error) {
	return auth.Resolve(flagServer, flagUsername, flagPassword)
}

// toServerInfo 将凭证转为 connector.ServerInfo
func toServerInfo(creds *auth.ServerCredentials) connector.ServerInfo {
	return connector.ServerInfo{
		URL:      creds.ServerURL,
		Username: creds.Username,
		Password: creds.Password,
		Token:    creds.Token,
	}
}

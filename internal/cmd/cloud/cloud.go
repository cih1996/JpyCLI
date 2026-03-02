package cloud

import (
	"github.com/spf13/cobra"
)

// NewCloudCmd 创建 cloud 主命令
func NewCloudCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cloud",
		Short: "集控平台远程 API 操作",
		Long: `操作集控平台的远程 API 接口。
包含设备管理、改机压力测试等功能。

使用前需要先配置密钥:
  jpy cloud config --server-url wss://home.accjs.cn/ws --secret-key <your_key>`,
	}

	cmd.AddCommand(NewConfigCmd())
	cmd.AddCommand(NewStressCmd())

	return cmd
}

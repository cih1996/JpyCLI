package flash

import (
	"github.com/spf13/cobra"
)

func NewFlashCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "flash",
		Short: "批量刷机工具",
		Long: `批量刷机工具，支持按 COM 口和通道批量刷机。

工作流程：
  1. 检查设备状态
  2. 发送 reboot bootloader
  3. 切换 COM 通道为 HUB 模式
  4. 执行刷机脚本
  5. 刷机成功后切换回 OTG 模式`,
	}

	cmd.AddCommand(newRunCmd())

	return cmd
}

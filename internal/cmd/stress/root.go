package stress

import (
	"cnb.cool/accbot/goTool/logs"
	"github.com/spf13/cobra"
)

func init() {
	// 在包初始化时就抑制 SDK 内部日志，避免干扰终端
	silentLogConfig := `
level: error
console: false
file: false
`
	_ = logs.SetLoggerConfig(silentLogConfig, 0)
}

// NewStressCmd 创建 stress 命令
func NewStressCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stress",
		Short: "压力测试命令组",
		Long:  `压力测试相关命令，包括用户端改机压力测试等。`,
	}

	cmd.AddCommand(newUserCmd())

	return cmd
}

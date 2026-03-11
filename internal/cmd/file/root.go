package file

import (
	"github.com/spf13/cobra"
)

func NewFileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "file",
		Short: "文件传输命令（上传/下载）",
	}

	cmd.AddCommand(newPushCmd())
	cmd.AddCommand(newPullCmd())

	return cmd
}

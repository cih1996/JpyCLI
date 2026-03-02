package server

import "github.com/spf13/cobra"

func NewServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "后台服务管理命令",
	}
	cmd.AddCommand(NewSSHServerCmd())
	return cmd
}

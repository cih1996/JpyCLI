package middleware

import (
	"github.com/spf13/cobra"
)

func NewMiddlewareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "middleware",
		Short: "中间件配置工具",
	}

	cmd.AddCommand(NewCreateCmd())
	return cmd
}

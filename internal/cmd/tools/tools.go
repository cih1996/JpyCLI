package tools

import (
	"jpy-cli/internal/cmd/tools/middleware"

	"github.com/spf13/cobra"
)

func NewToolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tools",
		Short: "实用工具集",
	}

	cmd.AddCommand(middleware.NewMiddlewareCmd())
	cmd.AddCommand(NewCompletionInstallCmd())
	return cmd
}

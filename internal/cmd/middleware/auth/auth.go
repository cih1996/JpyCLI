package auth

import (
	"github.com/spf13/cobra"
)

func NewAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "认证和服务器管理",
	}

	cmd.AddCommand(NewLoginCmd())
	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewTemplateCmd())
	cmd.AddCommand(NewImportCmd())
	cmd.AddCommand(NewSelectCmd())
	cmd.AddCommand(NewCreateCmd())
	cmd.AddCommand(NewExportCmd())

	return cmd
}

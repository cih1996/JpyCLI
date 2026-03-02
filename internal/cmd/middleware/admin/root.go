package admin

import (
	"github.com/spf13/cobra"
)

func NewAdminCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "管理员管理命令",
	}

	cmd.AddCommand(NewAutoAuthCmd())
	cmd.AddCommand(NewUpdateClusterCmd())

	return cmd
}

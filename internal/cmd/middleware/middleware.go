package middleware

import (
	"jpy-cli/internal/cmd/middleware/admin"
	"jpy-cli/internal/cmd/middleware/auth"
	"jpy-cli/internal/cmd/middleware/device"

	"github.com/spf13/cobra"
)

func NewMiddlewareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "middleware",
		Short: "中间件管理命令",
	}

	cmd.AddCommand(auth.NewAuthCmd())
	cmd.AddCommand(device.NewDeviceCmd())
	cmd.AddCommand(admin.NewAdminCmd())
	cmd.AddCommand(NewSSHCmd())
	cmd.AddCommand(NewRemoveCmd())
	cmd.AddCommand(NewReloginCmd())
	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewRestartCmd())

	return cmd
}

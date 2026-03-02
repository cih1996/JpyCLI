package device

import (
	"github.com/spf13/cobra"
)

func NewDeviceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "device",
		Short: "设备管理命令",
	}

	cmd.AddCommand(NewStatusCmd())
	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewExportCmd())
	cmd.AddCommand(NewRebootCmd())
	cmd.AddCommand(NewUSBCmd())
	cmd.AddCommand(NewADBCmd())
	cmd.AddCommand(NewLogCmd())

	return cmd
}

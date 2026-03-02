package middleware

import (
	"jpy-cli/internal/cmd/middleware/device"
	"jpy-cli/pkg/config"
	"jpy-cli/pkg/logger"
	"jpy-cli/pkg/middleware/device/controller"
	"jpy-cli/pkg/middleware/device/selector"
	"jpy-cli/pkg/middleware/model"

	"github.com/spf13/cobra"
)

func NewRestartCmd() *cobra.Command {
	opts := device.CommonFlags{}
	cmd := &cobra.Command{
		Use:   "restart",
		Short: "重启服务 (boxCore)",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Interactive = shouldEnterInteractive(cmd, &opts)

			return runRestartAction(opts, func(c *controller.DeviceController, devices []model.DeviceInfo) error {
				logger.Infof("已选择 %d 台设备，即将对所属中间件服务进行重启...", len(devices))
				// default service="boxCore", action=3
				return c.RestartServiceBatch(devices, "boxCore", 3)
			})
		},
	}
	device.AddCommonFlags(cmd, &opts)
	return cmd
}

func runRestartAction(opts device.CommonFlags, action func(*controller.DeviceController, []model.DeviceInfo) error) error {
	selOpts, err := opts.ToSelectorOptions()
	if err != nil {
		return err
	}

	devices, err := selector.SelectDevices(selOpts)
	if err != nil {
		return err
	}

	if len(devices) == 0 {
		logger.Warn("没有找到符合条件的设备。")
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctrl := controller.NewDeviceController(cfg)
	return action(ctrl, devices)
}

func shouldEnterInteractive(cmd *cobra.Command, opts *device.CommonFlags) bool {
	if opts.Interactive {
		return true
	}

	if opts.All {
		return false
	}

	filterFlags := []string{
		"group", "server", "uuid", "seat",
		"filter-adb", "filter-usb", "filter-online", "filter-has-ip",
		"authorized",
	}

	for _, name := range filterFlags {
		if cmd.Flags().Changed(name) {
			return false
		}
	}

	return true
}

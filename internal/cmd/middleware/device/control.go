package device

import (
	"fmt"
	"jpy-cli/pkg/config"
	"jpy-cli/pkg/logger"
	"jpy-cli/pkg/middleware/device/controller"
	"jpy-cli/pkg/middleware/device/selector"
	"jpy-cli/pkg/middleware/model"
	"jpy-cli/pkg/tui"
	"strings"

	"github.com/spf13/cobra"
)

type ControlOptions = CommonFlags

func NewRebootCmd() *cobra.Command {
	opts := CommonFlags{}
	cmd := &cobra.Command{
		Use:   "reboot",
		Short: "重启设备",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Interactive = shouldEnterInteractive(cmd, &opts)
			return runControlAction(opts, func(c *controller.DeviceController, devices []model.DeviceInfo) error {
				logger.Infof("正在重启 %d 台设备...", len(devices))
				return c.RebootBatch(devices, nil)
			})
		},
	}
	AddCommonFlags(cmd, &opts)
	return cmd
}

func NewUSBCmd() *cobra.Command {
	opts := CommonFlags{}
	var mode string
	cmd := &cobra.Command{
		Use:   "usb",
		Short: "切换USB模式 (host/device)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("mode") {
				options := []tui.Option{
					{Label: "Device (USB)", Value: "device"},
					{Label: "Host (OTG)", Value: "host"},
				}
				val, err := tui.SelectOption("请选择 USB 模式:", "", options)
				if err != nil {
					return err
				}
				mode = val
			}

			opts.Interactive = shouldEnterInteractive(cmd, &opts)

			otg := false
			if strings.ToLower(mode) == "host" || strings.ToLower(mode) == "otg" {
				otg = true
			} else if strings.ToLower(mode) == "device" || strings.ToLower(mode) == "usb" {
				otg = false
			} else {
				return fmt.Errorf("无效模式: %s (请使用 'host' 或 'device')", mode)
			}

			modeStr := "USB (Device)"
			if otg {
				modeStr = "OTG (Host)"
			}
			return runControlAction(opts, func(c *controller.DeviceController, devices []model.DeviceInfo) error {
				logger.Infof("正在将 %d 台设备切换至 %s 模式...", len(devices), modeStr)
				return c.SwitchUSBBatch(devices, otg, nil)
			})
		},
	}
	AddCommonFlags(cmd, &opts)
	cmd.Flags().StringVarP(&mode, "mode", "m", "device", "USB模式: 'host' (OTG) 或 'device' (USB)")
	return cmd
}

func NewADBCmd() *cobra.Command {
	opts := CommonFlags{}
	var state string
	cmd := &cobra.Command{
		Use:   "adb",
		Short: "控制ADB状态 (开启/关闭)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("set") {
				options := []tui.Option{
					{Label: "开启 (ON)", Value: "on"},
					{Label: "关闭 (OFF)", Value: "off"},
				}
				val, err := tui.SelectOption("请选择 ADB 状态:", "", options)
				if err != nil {
					return err
				}
				state = val
			}

			opts.Interactive = shouldEnterInteractive(cmd, &opts)

			enable := false
			if strings.ToLower(state) == "on" || strings.ToLower(state) == "true" {
				enable = true
			} else if strings.ToLower(state) == "off" || strings.ToLower(state) == "false" {
				enable = false
			} else {
				return fmt.Errorf("无效状态: %s (请使用 'on' 或 'off')", state)
			}

			actionStr := "关闭"
			if enable {
				actionStr = "开启"
			}
			return runControlAction(opts, func(c *controller.DeviceController, devices []model.DeviceInfo) error {
				logger.Infof("正在%s %d 台设备的ADB...", actionStr, len(devices))
				return c.ControlADBBatch(devices, enable, nil)
			})
		},
	}
	AddCommonFlags(cmd, &opts)
	cmd.Flags().StringVar(&state, "set", "off", "ADB状态: 'on' 或 'off'")
	return cmd
}

func runControlAction(opts CommonFlags, action func(*controller.DeviceController, []model.DeviceInfo) error) error {
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

func shouldEnterInteractive(cmd *cobra.Command, opts *CommonFlags) bool {
	// 1. 如果显式指定了交互模式，则使用
	if opts.Interactive {
		return true
	}

	// 2. 如果指定了 --all，且没有显式指定交互模式，则跳过交互模式
	if opts.All {
		return false
	}

	// 3. 检查是否有任何筛选参数被修改
	filterFlags := []string{
		"group", "server", "uuid", "seat",
		"filter-adb", "filter-usb", "filter-online", "filter-has-ip",
		"authorized",
	}

	for _, name := range filterFlags {
		if cmd.Flags().Changed(name) {
			// 如果提供了筛选条件，则不强制进入交互模式
			return false
		}
	}

	// 4. 如果没有筛选条件且没有指定 --all，强制进入交互模式
	return true
}

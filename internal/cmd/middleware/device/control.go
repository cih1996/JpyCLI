package device

import (
	"fmt"
	"jpy-cli/pkg/config"
	"jpy-cli/pkg/logger"
	"jpy-cli/pkg/middleware/device/controller"
	"jpy-cli/pkg/middleware/device/selector"
	"jpy-cli/pkg/middleware/model"
	"jpy-cli/pkg/tui"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type ControlOptions = CommonFlags

func NewRebootCmd() *cobra.Command {
	opts := CommonFlags{}
	cmd := &cobra.Command{
		Use:   "reboot",
		Short: "重启设备",
		Long: `重启设备。

输出模式:
  --output tui     交互式界面（默认）
  --output plain   纯文本输出
  --output json    JSON 格式输出`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Output == "plain" || opts.Output == "json" {
				return runControlCollect(cmd, opts, "reboot", func(c *controller.DeviceController, devices []model.DeviceInfo) ([]controller.BatchResult, error) {
					return c.RebootBatchCollect(devices, func(done, total int) {
						if opts.Output == "plain" {
							fmt.Fprintf(os.Stderr, "\r进度: %d/%d", done, total)
						}
					})
				})
			}
			// TUI mode
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
		Long: `切换USB模式。

输出模式:
  --output tui     交互式界面（默认）
  --output plain   纯文本输出
  --output json    JSON 格式输出

非交互模式必须指定 --mode 参数。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 非交互模式下必须指定 --mode
			if (opts.Output == "plain" || opts.Output == "json") && !cmd.Flags().Changed("mode") {
				return fmt.Errorf("非交互模式 (-o %s) 必须指定 --mode 参数 (host/device)", opts.Output)
			}

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

			otg := false
			if strings.ToLower(mode) == "host" || strings.ToLower(mode) == "otg" {
				otg = true
			} else if strings.ToLower(mode) == "device" || strings.ToLower(mode) == "usb" {
				otg = false
			} else {
				return fmt.Errorf("无效模式: %s (请使用 'host' 或 'device')", mode)
			}

			if opts.Output == "plain" || opts.Output == "json" {
				return runControlCollect(cmd, opts, "usb", func(c *controller.DeviceController, devices []model.DeviceInfo) ([]controller.BatchResult, error) {
					return c.SwitchUSBBatchCollect(devices, otg, func(done, total int) {
						if opts.Output == "plain" {
							fmt.Fprintf(os.Stderr, "\r进度: %d/%d", done, total)
						}
					})
				})
			}

			// TUI mode
			opts.Interactive = shouldEnterInteractive(cmd, &opts)
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
		Long: `控制ADB开启或关闭。

输出模式:
  --output tui     交互式界面（默认）
  --output plain   纯文本输出
  --output json    JSON 格式输出

非交互模式必须指定 --set 参数。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 非交互模式下必须指定 --set
			if (opts.Output == "plain" || opts.Output == "json") && !cmd.Flags().Changed("set") {
				return fmt.Errorf("非交互模式 (-o %s) 必须指定 --set 参数 (on/off)", opts.Output)
			}

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

			enable := false
			if strings.ToLower(state) == "on" || strings.ToLower(state) == "true" {
				enable = true
			} else if strings.ToLower(state) == "off" || strings.ToLower(state) == "false" {
				enable = false
			} else {
				return fmt.Errorf("无效状态: %s (请使用 'on' 或 'off')", state)
			}

			if opts.Output == "plain" || opts.Output == "json" {
				return runControlCollect(cmd, opts, "adb", func(c *controller.DeviceController, devices []model.DeviceInfo) ([]controller.BatchResult, error) {
					return c.ControlADBBatchCollect(devices, enable, func(done, total int) {
						if opts.Output == "plain" {
							fmt.Fprintf(os.Stderr, "\r进度: %d/%d", done, total)
						}
					})
				})
			}

			// TUI mode
			opts.Interactive = shouldEnterInteractive(cmd, &opts)
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

// runControlCollect 收集模式运行控制命令（用于 plain/json 输出）
func runControlCollect(cmd *cobra.Command, opts CommonFlags, action string,
	exec func(*controller.DeviceController, []model.DeviceInfo) ([]controller.BatchResult, error)) error {

	selOpts, err := opts.ToSelectorOptions()
	if err != nil {
		return err
	}
	// 非交互模式下强制关闭交互，跳过 BubbleTea 进度条
	selOpts.Interactive = false
	selOpts.Silent = true

	devices, err := selector.SelectDevices(selOpts)
	if err != nil {
		return err
	}

	if len(devices) == 0 {
		if opts.Output == "json" {
			fmt.Println("{\"total\":0,\"success\":0,\"failed\":0,\"results\":[]}")
			return nil
		}
		fmt.Println("没有找到符合条件的设备。")
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctrl := controller.NewDeviceController(cfg)
	results, err := exec(ctrl, devices)
	if err != nil {
		return err
	}

	if opts.Output == "plain" {
		fmt.Fprintln(os.Stderr) // 换行（接在进度条后面）
		printControlPlain(results)
	} else {
		printControlJSON(results)
	}

	// 设置退出码
	exitCode := controlExitCode(results)
	if exitCode > 0 {
		os.Exit(exitCode)
	}

	return nil
}

// runControlAction TUI 模式运行控制命令（原逻辑）
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
	if opts.Interactive {
		return true
	}

	// 非 TUI 输出模式强制跳过交互
	if opts.Output == "plain" || opts.Output == "json" {
		return false
	}

	if opts.All {
		return false
	}

	filterFlags := []string{
		"group", "server", "uuid", "seat", "ip",
		"filter-adb", "filter-usb", "filter-online", "filter-has-ip",
		"authorized",
		"set", "mode", // Allow action flags to trigger non-interactive mode
	}

	for _, name := range filterFlags {
		if cmd.Flags().Changed(name) {
			return false
		}
	}

	return true
}

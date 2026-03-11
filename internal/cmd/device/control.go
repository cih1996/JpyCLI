package device

import (
	"encoding/json"
	"fmt"
	"jpy-cli/pkg/middleware/connector"
	"jpy-cli/pkg/middleware/device/controller"
	"jpy-cli/pkg/middleware/device/selector"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func NewRebootCmd() *cobra.Command {
	var (
		filterIP   string
		filterUUID string
		filterSeat int
	)

	cmd := &cobra.Command{
		Use:   "reboot",
		Short: "重启设备",
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := resolveCredentials()
			if err != nil {
				return err
			}
			server := toServerInfo(creds)

			devices, err := selector.SelectDevices(selector.SelectionOptions{
				Servers: []connector.ServerInfo{server},
				IP:      filterIP,
				UUID:    filterUUID,
				Seat:    filterSeat,
			})
			if err != nil {
				return err
			}
			if len(devices) == 0 {
				return fmt.Errorf("没有匹配的设备")
			}

			ctrl := controller.NewSingleServerController(server)
			results, err := ctrl.RebootBatchCollect(devices, nil)
			if err != nil {
				return err
			}

			return printControlResults(results, flagOutput)
		},
	}

	cmd.Flags().StringVar(&filterIP, "ip", "", "按设备 IP 过滤")
	cmd.Flags().StringVar(&filterUUID, "uuid", "", "按设备 UUID 过滤")
	cmd.Flags().IntVar(&filterSeat, "seat", -1, "按机位号过滤")

	return cmd
}

func NewUSBCmd() *cobra.Command {
	var (
		filterIP   string
		filterUUID string
		filterSeat int
		mode       string
	)

	cmd := &cobra.Command{
		Use:   "usb",
		Short: "切换 USB 模式 (host/device)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if mode == "" {
				return fmt.Errorf("必须指定 --mode 参数 (host/device)")
			}

			otg := false
			switch strings.ToLower(mode) {
			case "host", "otg":
				otg = true
			case "device", "usb":
				otg = false
			default:
				return fmt.Errorf("无效模式: %s (请使用 host 或 device)", mode)
			}

			creds, err := resolveCredentials()
			if err != nil {
				return err
			}
			server := toServerInfo(creds)

			devices, err := selector.SelectDevices(selector.SelectionOptions{
				Servers: []connector.ServerInfo{server},
				IP:      filterIP,
				UUID:    filterUUID,
				Seat:    filterSeat,
			})
			if err != nil {
				return err
			}
			if len(devices) == 0 {
				return fmt.Errorf("没有匹配的设备")
			}

			ctrl := controller.NewSingleServerController(server)
			results, err := ctrl.SwitchUSBBatchCollect(devices, otg, nil)
			if err != nil {
				return err
			}

			return printControlResults(results, flagOutput)
		},
	}

	cmd.Flags().StringVar(&filterIP, "ip", "", "按设备 IP 过滤")
	cmd.Flags().StringVar(&filterUUID, "uuid", "", "按设备 UUID 过滤")
	cmd.Flags().IntVar(&filterSeat, "seat", -1, "按机位号过滤")
	cmd.Flags().StringVarP(&mode, "mode", "m", "", "USB 模式: host/device（必填）")

	return cmd
}

func NewADBCmd() *cobra.Command {
	var (
		filterIP   string
		filterUUID string
		filterSeat int
		state      string
	)

	cmd := &cobra.Command{
		Use:   "adb",
		Short: "控制 ADB 状态 (on/off)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if state == "" {
				return fmt.Errorf("必须指定 --set 参数 (on/off)")
			}

			enable := false
			switch strings.ToLower(state) {
			case "on", "true":
				enable = true
			case "off", "false":
				enable = false
			default:
				return fmt.Errorf("无效状态: %s (请使用 on 或 off)", state)
			}

			creds, err := resolveCredentials()
			if err != nil {
				return err
			}
			server := toServerInfo(creds)

			devices, err := selector.SelectDevices(selector.SelectionOptions{
				Servers: []connector.ServerInfo{server},
				IP:      filterIP,
				UUID:    filterUUID,
				Seat:    filterSeat,
			})
			if err != nil {
				return err
			}
			if len(devices) == 0 {
				return fmt.Errorf("没有匹配的设备")
			}

			ctrl := controller.NewSingleServerController(server)
			results, err := ctrl.ControlADBBatchCollect(devices, enable, nil)
			if err != nil {
				return err
			}

			return printControlResults(results, flagOutput)
		},
	}

	cmd.Flags().StringVar(&filterIP, "ip", "", "按设备 IP 过滤")
	cmd.Flags().StringVar(&filterUUID, "uuid", "", "按设备 UUID 过滤")
	cmd.Flags().IntVar(&filterSeat, "seat", -1, "按机位号过滤")
	cmd.Flags().StringVar(&state, "set", "", "ADB 状态: on/off（必填）")

	return cmd
}

func printControlResults(results []controller.BatchResult, outputMode string) error {
	success := 0
	failed := 0
	for _, r := range results {
		if r.OK {
			success++
		} else {
			failed++
		}
	}

	switch outputMode {
	case "json":
		type jsonResult struct {
			Total   int                      `json:"total"`
			Success int                      `json:"success"`
			Failed  int                      `json:"failed"`
			Results []controller.BatchResult `json:"results"`
		}
		b, _ := json.Marshal(jsonResult{
			Total: len(results), Success: success, Failed: failed, Results: results,
		})
		fmt.Println(string(b))
	default:
		for _, r := range results {
			status := "OK"
			if !r.OK {
				status = "FAIL:" + r.Error
			}
			fmt.Printf("%s\t%d\t%s\t%s\n", r.Server, r.Seat, r.UUID, status)
		}
		fmt.Fprintf(os.Stderr, "--- total: %d, success: %d, failed: %d\n", len(results), success, failed)
	}

	if failed > 0 {
		os.Exit(1)
	}
	return nil
}

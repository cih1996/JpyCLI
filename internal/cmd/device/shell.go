package device

import (
	"fmt"
	"jpy-cli/pkg/middleware/connector"
	"jpy-cli/pkg/middleware/device/api"
	"jpy-cli/pkg/middleware/device/selector"
	"strings"

	"github.com/spf13/cobra"
)

func NewShellCmd() *cobra.Command {
	var (
		deviceIP string
		command  string
		seat     int
	)

	cmd := &cobra.Command{
		Use:   "shell [command]",
		Short: "向设备发送 shell 命令",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && command == "" {
				command = args[0]
			}
			if command == "" {
				return fmt.Errorf("必须指定要执行的命令")
			}
			if deviceIP == "" && seat < 0 {
				return fmt.Errorf("必须指定 --ip 或 --seat")
			}

			creds, err := resolveCredentials()
			if err != nil {
				return err
			}
			server := toServerInfo(creds)

			// 通过 IP 查找 seat
			targetSeat := seat
			if deviceIP != "" {
				devices, err := selector.SelectDevices(selector.SelectionOptions{
					Servers: []connector.ServerInfo{server},
					Seat:    -1,
				})
				if err != nil {
					return fmt.Errorf("获取设备列表失败: %v", err)
				}
				matched := false
				for _, d := range devices {
					if strings.Contains(d.IP, deviceIP) {
						targetSeat = d.Seat
						matched = true
						break
					}
				}
				if !matched {
					return fmt.Errorf("未找到 IP 为 %s 的设备", deviceIP)
				}
			}

			// 尝试 f=14
			result, err := shellExecViaF14(server, targetSeat, command)
			if err == nil {
				return shellOutputResult(result, server.URL, targetSeat, command, flagOutput)
			}

			// 降级 f=289
			ws2, err2 := connector.ConnectGuard(server)
			if err2 != nil {
				return fmt.Errorf("连接失败: %v", err2)
			}
			defer ws2.Close()
			deviceAPI := api.NewDeviceAPI(ws2, server.URL, server.Token)

			result, err = deviceAPI.ExecuteShell(targetSeat, command)
			if err != nil {
				if strings.Contains(err.Error(), "code 10") {
					return shellExecViaTerminal(server, targetSeat, command, flagOutput)
				}
				return fmt.Errorf("执行命令失败: %v", err)
			}

			return shellOutputResult(result, server.URL, targetSeat, command, flagOutput)
		},
	}

	cmd.Flags().StringVar(&deviceIP, "ip", "", "目标设备 IP")
	cmd.Flags().StringVarP(&command, "command", "c", "", "要执行的 shell 命令")
	cmd.Flags().IntVar(&seat, "seat", -1, "设备机位号")

	return cmd
}

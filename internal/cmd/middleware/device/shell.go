package device

import (
	"encoding/json"
	"fmt"
	"jpy-cli/pkg/config"
	"jpy-cli/pkg/middleware/connector"
	"jpy-cli/pkg/middleware/device/api"
	"jpy-cli/pkg/middleware/device/terminal"
	"jpy-cli/pkg/middleware/model"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func NewShellCmd() *cobra.Command {
	var (
		serverPattern string
		deviceIP      string
		command       string
		output        string
		seat          int
	)

	cmd := &cobra.Command{
		Use:   "shell",
		Short: "向设备发送 shell 命令",
		Long: `通过中间件向指定设备发送 shell 命令并返回输出。

示例:
  # 让设备重启到 fastboot 模式
  jpy middleware device shell --server 192.168.255.1 --ip 192.168.10.195 --command "reboot bootloader"

  # JSON 输出（AI/脚本解析）
  jpy middleware device shell --server 192.168.255.1 --ip 192.168.10.195 -c "getprop ro.product.model" -o json

  # 直接通过机位号定位设备
  jpy middleware device shell --server 192.168.255.1 --seat 3 -c "reboot bootloader"

输出模式:
  --output plain   纯文本（默认），直接输出命令结果
  --output json    JSON 格式，包含 server/seat/command/output/exit_code
  --output tui     同 plain`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 参数校验
			if serverPattern == "" {
				return fmt.Errorf("必须指定 --server 参数（中间件服务器地址）")
			}
			if command == "" {
				return fmt.Errorf("必须指定 --command（或 -c）参数")
			}
			if deviceIP == "" && seat < 0 {
				return fmt.Errorf("必须指定 --ip（设备IP）或 --seat（机位号）之一")
			}

			// 1. 加载配置
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("加载配置失败: %v", err)
			}

			// 2. 查找服务器配置（模糊匹配 URL）
			serverCfg, found := shellFindServer(cfg, serverPattern)
			if !found {
				return fmt.Errorf("未找到匹配的服务器: %s（请先通过 middleware auth login 添加）", serverPattern)
			}

			// 3. 连接 Guard WebSocket
			connSvc := connector.NewConnectorService(cfg)
			ws, err := connSvc.ConnectGuard(serverCfg)
			if err != nil {
				return fmt.Errorf("连接服务器 %s 失败: %v", serverCfg.URL, err)
			}
			defer ws.Close()

			deviceAPI := api.NewDeviceAPI(ws, serverCfg.URL, serverCfg.Token)

			// 4. 通过 IP 查找机位号（如果未直接提供 --seat）
			targetSeat := seat
			if deviceIP != "" {
				statuses, err := deviceAPI.FetchOnlineStatus()
				if err != nil {
					return fmt.Errorf("获取设备在线状态失败: %v", err)
				}
				matched := false
				for _, s := range statuses {
					if s.IP == deviceIP {
						targetSeat = s.Seat
						matched = true
						break
					}
				}
				if !matched {
					return fmt.Errorf("在服务器 %s 上未找到 IP 为 %s 的设备", serverCfg.URL, deviceIP)
				}
			}

			// 5. 执行 shell 命令
			if output != "json" {
				fmt.Fprintf(os.Stderr, "[%s] 机位 %d 执行: %s\n", serverCfg.URL, targetSeat, command)
			}
			result, err := deviceAPI.ExecuteShell(targetSeat, command)
			if err != nil {
				// code 10 = 老系统不支持该功能（unsupported function），自动降级到 Terminal 通道
				if strings.Contains(err.Error(), "code 10") {
					return shellExecViaTerminal(connSvc, serverCfg, targetSeat, command, output)
				}
				if output == "json" {
					type errOut struct {
						Server  string `json:"server"`
						Seat    int    `json:"seat"`
						Command string `json:"command"`
						Success bool   `json:"success"`
						Error   string `json:"error"`
					}
					b, _ := json.Marshal(errOut{
						Server:  serverCfg.URL,
						Seat:    targetSeat,
						Command: command,
						Success: false,
						Error:   err.Error(),
					})
					fmt.Println(string(b))
					os.Exit(1)
				}
				return fmt.Errorf("执行命令失败: %v", err)
			}

			// 6. 输出结果
			return shellOutputResult(result, serverCfg.URL, targetSeat, command, output)
		},
	}

	cmd.Flags().StringVarP(&serverPattern, "server", "s", "", "中间件服务器地址（IP 或 URL 关键词，必填）")
	cmd.Flags().StringVar(&deviceIP, "ip", "", "目标设备 IP 地址（与 --seat 二选一）")
	cmd.Flags().StringVarP(&command, "command", "c", "", "要执行的 shell 命令（必填）")
	cmd.Flags().StringVarP(&output, "output", "o", "plain", "输出模式: plain/json/tui")
	cmd.Flags().IntVar(&seat, "seat", -1, "设备机位号（与 --ip 二选一）")

	return cmd
}

// shellFindServer 根据 pattern 在配置中查找服务器（模糊匹配 URL）
func shellFindServer(cfg *config.Config, pattern string) (config.LocalServerConfig, bool) {
	for _, servers := range cfg.Groups {
		for _, s := range servers {
			if s.Disabled {
				continue
			}
			if strings.Contains(s.URL, pattern) {
				return s, true
			}
		}
	}
	return config.LocalServerConfig{}, false
}

// shellExecViaTerminal 通过 Terminal 通道发送 shell 命令（老系统兼容降级方案）。
// 老系统不支持 f=289（FuncCMDWithResult，code 10），但支持 f=9（Terminal Init）。
// Terminal 模式无法获取命令返回值，适合 reboot bootloader 等"发送即可"的指令。
func shellExecViaTerminal(connSvc *connector.ConnectorService, serverCfg config.LocalServerConfig, seat int, command, outputMode string) error {
	if outputMode != "json" {
		fmt.Fprintf(os.Stderr, "⚠️  老系统不支持 shell API (code 10)，切换到 Terminal 通道...\n")
	}

	// 建立 Terminal 专用连接（Guard endpoint，id=seat）
	ws, err := connSvc.ConnectDeviceTerminal(serverCfg, int64(seat))
	if err != nil {
		return fmt.Errorf("Terminal 通道连接失败: %v", err)
	}
	defer ws.Close()

	term := terminal.NewTerminalSession(ws, int64(seat))

	// 初始化 Terminal（发送 f=9 握手）
	if err := term.Init(); err != nil {
		return fmt.Errorf("Terminal 初始化失败: %v", err)
	}

	// 等待 shell 就绪（出现 $ 或 # 提示符），超时后仍尝试发送
	if err := term.WaitForReady(8 * time.Second); err != nil {
		if outputMode != "json" {
			fmt.Fprintf(os.Stderr, "⚠️  等待 Terminal 就绪超时，仍然尝试发送命令...\n")
		}
	}

	// 发送命令（Terminal 模式无结构化返回值）
	if err := term.Exec(command); err != nil {
		return fmt.Errorf("Terminal 发送命令失败: %v", err)
	}

	// 等待一小段时间确保命令已写入（reboot 类命令会立即断连）
	time.Sleep(500 * time.Millisecond)

	switch outputMode {
	case "json":
		type shellJSON struct {
			Server  string `json:"server"`
			Seat    int    `json:"seat"`
			Command string `json:"command"`
			Output  string `json:"output"`
			Success bool   `json:"success"`
			Note    string `json:"note"`
		}
		b, _ := json.Marshal(shellJSON{
			Server:  serverCfg.URL,
			Seat:    seat,
			Command: command,
			Output:  "",
			Success: true,
			Note:    "老系统兼容模式：通过 Terminal 通道发送，无返回值",
		})
		fmt.Println(string(b))
	default:
		fmt.Fprintf(os.Stderr, "✓ 命令已通过 Terminal 通道发送（老系统兼容，无返回值）\n")
	}

	return nil
}

// shellOutputResult 输出 shell 执行结果
func shellOutputResult(result *model.ShellResult, server string, seat int, command string, outputMode string) error {
	switch outputMode {
	case "json":
		type shellJSON struct {
			Server   string  `json:"server"`
			Seat     int     `json:"seat"`
			Command  string  `json:"command"`
			Output   string  `json:"output"`
			ExitCode *int    `json:"exit_code,omitempty"`
			Error    *string `json:"error,omitempty"`
			Success  bool    `json:"success"`
		}
		out := shellJSON{
			Server:   server,
			Seat:     seat,
			Command:  command,
			Output:   result.Output,
			ExitCode: result.ExitCode,
			Error:    result.Error,
			Success:  result.Error == nil || *result.Error == "",
		}
		b, err := json.Marshal(out)
		if err != nil {
			return err
		}
		fmt.Println(string(b))
		// 按退出码设置进程退出码
		if result.ExitCode != nil && *result.ExitCode != 0 {
			os.Exit(*result.ExitCode)
		}
	default: // plain / tui
		if result.Error != nil && *result.Error != "" {
			fmt.Fprintf(os.Stderr, "执行错误: %s\n", *result.Error)
		}
		if result.Output != "" {
			fmt.Print(result.Output)
			// 确保输出以换行结束
			if len(result.Output) > 0 && result.Output[len(result.Output)-1] != '\n' {
				fmt.Println()
			}
		}
		if result.ExitCode != nil && *result.ExitCode != 0 {
			os.Exit(*result.ExitCode)
		}
	}
	return nil
}

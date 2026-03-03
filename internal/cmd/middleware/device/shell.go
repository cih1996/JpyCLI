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
		Use:   "shell [command]",
		Short: "向设备发送 shell 命令（支持老系统）",
		Long: `通过中间件向指定设备发送 shell 命令并返回输出。

命令可作为位置参数（推荐），或通过 --command/-c 传入：
  jpy middleware device shell "ls -lh" --server 192.168.255.1 --ip 192.168.10.195
  jpy middleware device shell -s 192.168.255.1 --ip 192.168.10.195 -c "ls -lh"

参数说明：
  --server / -s   中间件服务器地址（IP 或 URL 关键词），指定与哪台中间件通信
  --ip            目标设备在中间件内的 IP（非中间件服务器 IP）
  --seat          目标设备机位号（与 --ip 二选一）

执行通道（自动选择，逐级降级）：
  1. f=14  设备连接通道（老/新系统通用，有返回值）
  2. f=289 Guard 管理通道（新系统，有返回值，老系统返回 code 10 自动降级）
  3. Terminal 通道（老系统兜底，无结构化返回值）

示例:
  # 查看设备文件列表（命令作位置参数）
  jpy middleware device shell "ls -lh" --server 192.168.255.1 --ip 192.168.10.195

  # 让设备重启到 fastboot 模式
  jpy middleware device shell "reboot bootloader" -s 192.168.255.1 --ip 192.168.10.195

  # JSON 输出（适合脚本解析）
  jpy middleware device shell "getprop ro.product.model" -s 192.168.255.1 --ip 192.168.10.195 -o json

  # 通过机位号定位设备
  jpy middleware device shell "reboot bootloader" -s 192.168.255.1 --seat 3

输出模式:
  --output plain   纯文本（默认），直接输出命令结果
  --output json    JSON 格式，包含 server/seat/command/output/exit_code
  --output tui     同 plain`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// 位置参数优先，其次 --command/-c
			if len(args) > 0 && command == "" {
				command = args[0]
			}

			// 参数校验
			if serverPattern == "" {
				return fmt.Errorf("必须指定 --server / -s 参数（中间件服务器地址）")
			}
			if command == "" {
				return fmt.Errorf("必须指定要执行的命令（位置参数或 --command / -c）")
			}
			if deviceIP == "" && seat < 0 {
				return fmt.Errorf("必须指定 --ip（目标设备 IP）或 --seat（机位号）之一")
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

			// 3. 连接 Guard WebSocket（用于获取设备列表）
			connSvc := connector.NewConnectorService(cfg)
			ws, err := connSvc.ConnectGuard(serverCfg)
			if err != nil {
				return fmt.Errorf("连接服务器 %s 失败: %v", serverCfg.URL, err)
			}

			deviceAPI := api.NewDeviceAPI(ws, serverCfg.URL, serverCfg.Token)

			// 4. 通过 IP 查找机位号（如果未直接提供 --seat）
			targetSeat := seat
			if deviceIP != "" {
				statuses, err := deviceAPI.FetchOnlineStatus()
				if err != nil {
					ws.Close()
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
					ws.Close()
					return fmt.Errorf("在服务器 %s 上未找到 IP 为 %s 的设备", serverCfg.URL, deviceIP)
				}
			}
			ws.Close() // 关闭查询连接，后续执行使用独立连接

			if output != "json" {
				fmt.Fprintf(os.Stderr, "[%s] 机位 %d 执行: %s\n", serverCfg.URL, targetSeat, command)
			}

			// 5. 优先尝试 f=14（设备连接通道，老/新系统通用）
			result, err := shellExecViaF14(connSvc, serverCfg, targetSeat, command)
			if err == nil {
				return shellOutputResult(result, serverCfg.URL, targetSeat, command, output)
			}
			if output != "json" {
				fmt.Fprintf(os.Stderr, "⚠️  f=14 通道失败（%v），降级到 f=289...\n", err)
			}

			// 6. 降级到 f=289（Guard 管理通道，新系统）
			ws2, err2 := connSvc.ConnectGuard(serverCfg)
			if err2 != nil {
				return fmt.Errorf("连接服务器 %s 失败: %v", serverCfg.URL, err2)
			}
			defer ws2.Close()
			deviceAPI2 := api.NewDeviceAPI(ws2, serverCfg.URL, serverCfg.Token)

			result, err = deviceAPI2.ExecuteShell(targetSeat, command)
			if err != nil {
				// code 10 = 老系统不支持（f=289），自动降级到 Terminal 通道
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

			return shellOutputResult(result, serverCfg.URL, targetSeat, command, output)
		},
	}

	cmd.Flags().StringVarP(&serverPattern, "server", "s", "", "中间件服务器地址（IP 或 URL 关键词，必填）")
	cmd.Flags().StringVar(&deviceIP, "ip", "", "目标设备 IP 地址（与 --seat 二选一）")
	cmd.Flags().StringVarP(&command, "command", "c", "", "要执行的 shell 命令（可用位置参数代替）")
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

// shellExecViaF14 通过 f=14（FuncBatchCommand）在设备专用连接上执行 shell 命令。
// 连接方式：/box/guard?id=<seat>，data 字段为命令字符串，无需在消息中指定 seat。
// 老系统和新系统均支持此接口，是首选执行通道。
func shellExecViaF14(connSvc *connector.ConnectorService, serverCfg config.LocalServerConfig, seat int, command string) (*model.ShellResult, error) {
	ws, err := connSvc.ConnectDeviceTerminal(serverCfg, int64(seat))
	if err != nil {
		return nil, fmt.Errorf("f=14 设备连接失败: %v", err)
	}
	defer ws.Close()

	resp, err := ws.SendRequest(model.FuncBatchCommand, command)
	if err != nil {
		return nil, fmt.Errorf("f=14 发送失败: %v", err)
	}
	if resp.Code != nil && *resp.Code != 0 {
		msg := "unknown"
		if resp.Msg != nil {
			msg = *resp.Msg
		}
		return nil, fmt.Errorf("f=14 执行失败 (code %d): %s", *resp.Code, msg)
	}

	// data 字段是 shell 输出字符串
	output := ""
	switch v := resp.Data.(type) {
	case string:
		output = v
	case []byte:
		output = string(v)
	}

	return &model.ShellResult{Output: output}, nil
}

// shellExecViaTerminal 通过 Terminal 通道发送 shell 命令（老系统兼容降级方案）。
// 老系统不支持 f=289（FuncCMDWithResult，code 10），但支持 f=9（Terminal Init）。
// Terminal 模式无法获取命令返回值，适合 reboot bootloader 等"发送即可"的指令。
func shellExecViaTerminal(connSvc *connector.ConnectorService, serverCfg config.LocalServerConfig, seat int, command, outputMode string) error {
	if outputMode != "json" {
		fmt.Fprintf(os.Stderr, "⚠️  降级到 Terminal 通道（无返回值）...\n")
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

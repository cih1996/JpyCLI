package device

import (
	"encoding/json"
	"fmt"
	"jpy-cli/pkg/config"
	"jpy-cli/pkg/middleware/connector"
	"jpy-cli/pkg/middleware/device/api"
	"jpy-cli/pkg/middleware/device/selector"
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

			// 3. 连接服务，准备后续执行
			connSvc := connector.NewConnectorService(cfg)

			// 4. 通过 IP 查找机位号（复用 selector，与 device list 同一数据源）
			targetSeat := seat
			if deviceIP != "" {
				if output != "json" {
					fmt.Fprintf(os.Stderr, "正在查询设备 IP %s 对应的机位号...\n", deviceIP)
				}
				selOpts := selector.SelectionOptions{
					ServerPattern: serverPattern,
					Silent:        true,
					Seat:          -1, // -1 表示不过滤机位号
				}
				allDevices, err := selector.SelectDevices(selOpts)
				if err != nil {
					return fmt.Errorf("获取设备列表失败: %v", err)
				}
				matched := false
				for _, d := range allDevices {
					if strings.Contains(d.IP, deviceIP) {
						targetSeat = d.Seat
						matched = true
						break
					}
				}
				if !matched {
					// 列出所有可用设备 IP，帮助用户确认正确地址
					var available []string
					for _, d := range allDevices {
						if d.IP != "" {
							available = append(available, fmt.Sprintf("seat=%d ip=%s", d.Seat, d.IP))
						}
					}
					if len(available) > 0 {
						return fmt.Errorf("在服务器 %s 上未找到 IP 为 %s 的设备\n可用设备列表:\n  %s",
							serverCfg.URL, deviceIP, strings.Join(available, "\n  "))
					}
					return fmt.Errorf("在服务器 %s 上未找到 IP 为 %s 的设备（设备列表为空，请先确认设备在线）",
						serverCfg.URL, deviceIP)
				}
			}

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

// shellExecViaF14 通过 f=14（FuncBatchCommand）向指定设备发送 shell 命令并获取输出。
// 连接方式：/box/guard?id=0（主 Guard 连接），协议 header 的 deviceIds 字段写入 seat 号，
// 这与切换 OTG/HUB 使用的是同一条 WebSocket 连接和协议层。
func shellExecViaF14(connSvc *connector.ConnectorService, serverCfg config.LocalServerConfig, seat int, command string) (*model.ShellResult, error) {
	ws, err := connSvc.ConnectGuard(serverCfg)
	if err != nil {
		return nil, fmt.Errorf("f=14 Guard 连接失败: %v", err)
	}
	defer ws.Close()

	// deviceIds header 写入 seat，data 字段是命令字符串
	resp, err := ws.SendRequestToDevice(model.FuncBatchCommand, command, uint64(seat))
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

// shellExecViaTerminal 通过 Terminal 通道执行 shell 命令并收集输出（老系统兼容）。
// 老系统不支持 f=14/f=289，但支持 f=9（Terminal Init）。
// Terminal 输出为 VT100 格式，本函数会等待命令完成并清理 ANSI 转义码。
func shellExecViaTerminal(connSvc *connector.ConnectorService, serverCfg config.LocalServerConfig, seat int, command, outputMode string) error {
	if outputMode != "json" {
		fmt.Fprintf(os.Stderr, "⚠️  降级到 Terminal 通道（VT100 模式）...\n")
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

	// 等待 shell 就绪（出现 $ 或 # 提示符）
	if err := term.WaitForReady(8 * time.Second); err != nil {
		if outputMode != "json" {
			fmt.Fprintf(os.Stderr, "⚠️  等待 Terminal 就绪超时，仍然尝试发送命令...\n")
		}
	}

	// 排空就绪前积累的输出
	for {
		select {
		case <-term.Output:
		default:
			goto Flushed
		}
	}
Flushed:

	// 发送命令
	if err := term.Exec(command); err != nil {
		return fmt.Errorf("Terminal 发送命令失败: %v", err)
	}

	// 收集命令输出，直到再次看到提示符（$ / #）或超时（15s）
	var buf strings.Builder
	deadline := time.After(15 * time.Second)
	lastChunk := time.Now()
	for {
		select {
		case chunk := <-term.Output:
			buf.WriteString(chunk)
			lastChunk = time.Now()
			// 检测提示符出现表示命令已执行完毕
			cleaned := termStripANSI(buf.String())
			if termHasPrompt(cleaned) {
				goto CollectDone
			}
		case <-deadline:
			goto CollectDone
		default:
			// 无新数据且距上次收到数据超过 1.5s，认为命令完成
			if time.Since(lastChunk) > 1500*time.Millisecond && buf.Len() > 0 {
				goto CollectDone
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
CollectDone:

	rawOutput := termStripANSI(buf.String())
	// 去掉命令回显和最后一行提示符
	rawOutput = termStripEcho(rawOutput, command)

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
		note := ""
		if rawOutput == "" {
			note = "Terminal 通道：命令已发送，无输出或输出超时"
		}
		b, _ := json.Marshal(shellJSON{
			Server:  serverCfg.URL,
			Seat:    seat,
			Command: command,
			Output:  rawOutput,
			Success: true,
			Note:    note,
		})
		fmt.Println(string(b))
	default:
		if rawOutput != "" {
			fmt.Print(rawOutput)
			if len(rawOutput) > 0 && rawOutput[len(rawOutput)-1] != '\n' {
				fmt.Println()
			}
		} else {
			fmt.Fprintf(os.Stderr, "✓ 命令已发送（Terminal 通道，无可解析输出）\n")
		}
	}

	return nil
}

// termStripANSI 去除 VT100/ANSI 转义序列（\x1b[...m 及 \r 等）
func termStripANSI(s string) string {
	var out strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			// 跳过 ESC [ ... 直到遇到字母
			i += 2
			for i < len(s) && (s[i] < 'A' || s[i] > 'Z') && (s[i] < 'a' || s[i] > 'z') {
				i++
			}
			if i < len(s) {
				i++ // 跳过终止字母
			}
		} else if s[i] == '\r' {
			i++
		} else {
			out.WriteByte(s[i])
			i++
		}
	}
	return out.String()
}

// termHasPrompt 检测字符串末尾是否出现 shell 提示符（$ 或 #）
func termHasPrompt(s string) bool {
	// 找最后一行非空内容
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		// 末尾是 $ 或 # 认为是提示符
		if strings.HasSuffix(line, "$") || strings.HasSuffix(line, "#") ||
			strings.Contains(line, "$ ") || strings.Contains(line, "# ") {
			return true
		}
		break
	}
	return false
}

// termStripEcho 去除命令回显行和最后的 shell 提示符行
func termStripEcho(output, command string) string {
	lines := strings.Split(output, "\n")
	var result []string
	echoSkipped := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		// 跳过命令回显行
		if !echoSkipped && strings.Contains(line, command) {
			echoSkipped = true
			continue
		}
		// 跳过末尾提示符行
		if termHasPrompt(trimmed) && trimmed != "" {
			continue
		}
		result = append(result, line)
	}
	// 去除头尾空行
	return strings.TrimSpace(strings.Join(result, "\n"))
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

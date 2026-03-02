package device

import (
	"fmt"
	wsclient "jpy-cli/pkg/client/ws"
	"jpy-cli/pkg/config"
	"jpy-cli/pkg/middleware/connector"
	"jpy-cli/pkg/middleware/device/selector"
	"jpy-cli/pkg/middleware/device/terminal"
	"jpy-cli/pkg/middleware/model"
	"strings"

	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

func NewLogCmd() *cobra.Command {
	var (
		serverFilter   string
		groupFilter    string
		uuidFilter     string
		seatFilter     int
		interactive    bool
		authorizedOnly string
	)

	cmd := &cobra.Command{
		Use:   "log",
		Short: "查看设备日志 (USB/ADB/Shell Flow)",
		Long: `自动执行以下流程查看设备日志：
1. 切换设备到 USB 模式 (Host)
2. 开启 ADB
3. 连接设备终端
4. 执行 tail -100 /data/local/tmp/guard/logs/guard.log
5. 实时显示输出

注意：此命令必须操作单个设备。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. Select Device
			// Note: SelectDevices loads config internally, but we need config later for connection.
			// So we load it here too or reuse if possible.
			// selector.SelectDevices loads it.

			// We need to convert seatFilter to int pointer if > 0, or handle 0 as "any"?
			// selector.SelectionOptions uses int for Seat (-1 for any usually, or 0? TS SDK uses -1? Selector uses int)
			// Let's check selector.SelectionOptions: Seat int.
			// Check logic: if opts.Seat > 0 ...

			var authPtr *bool
			if authorizedOnly != "" {
				val := authorizedOnly == "true"
				authPtr = &val
			}

			devices, err := selector.SelectDevices(selector.SelectionOptions{
				Group:          groupFilter,
				ServerPattern:  serverFilter,
				UUID:           uuidFilter,
				Seat:           seatFilter,
				Interactive:    interactive,
				AuthorizedOnly: authPtr,
			})
			if err != nil {
				return err
			}

			if len(devices) == 0 {
				return fmt.Errorf("未找到匹配设备")
			}

			// Handle multiple devices selection
			if len(devices) > 1 {
				// Use the shared interactive selector which provides a nice table view
				selected, err := selector.RunInteractiveSelection(devices)
				if err != nil {
					return err
				}

				if len(selected) != 1 {
					return fmt.Errorf("只能选择一台设备进行日志查看")
				}
				devices = selected
			}

			target := devices[0]

			// Show detailed info before confirmation
			status := "Offline"
			if target.IsOnline {
				status = "Online"
			}

			fmt.Println("\n即将连接设备:")
			fmt.Printf("  UUID:    %s\n", target.UUID)
			fmt.Printf("  Seat:    %d\n", target.Seat)
			fmt.Printf("  IP:      %s\n", target.IP)
			fmt.Printf("  Status:  %s\n", status)
			fmt.Printf("  Server:  %s\n", target.ServerURL)
			fmt.Println("\n此操作将重置 USB/ADB 状态...")

			// Load config for connection
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			// 2. Prepare (USB -> ADB)
			connService := connector.NewConnectorService(cfg)

			// Find Server Config
			var serverConfig config.LocalServerConfig
			found := false

			// Helper to find server
			for _, group := range cfg.Groups {
				for _, s := range group {
					if s.URL == target.ServerURL {
						serverConfig = s
						found = true
						break
					}
				}
				if found {
					break
				}
			}

			if !found {
				return fmt.Errorf("未找到服务器配置: %s", target.ServerURL)
			}

			// Step 2.1: Switch to USB (Host)
			fmt.Println("\n[1/4] 正在切换到 USB 模式 (Host)...")
			wsGuard, err := connService.ConnectGuard(serverConfig)
			if err != nil {
				return fmt.Errorf("连接 Guard 失败: %v", err)
			}

			// SwitchUSBMode
			if err := api_SwitchUSBMode(wsGuard, target.Seat, false); err != nil {
				wsGuard.Close()
				return fmt.Errorf("切换 USB 模式失败: %v", err)
			}

			fmt.Println("等待 5 秒...")
			time.Sleep(5 * time.Second)

			// Step 2.2: Enable ADB
			fmt.Println("[2/4] 正在开启 ADB...")
			if err := api_ControlADB(wsGuard, target.Seat, true); err != nil {
				wsGuard.Close()
				return fmt.Errorf("开启 ADB 失败: %v", err)
			}
			wsGuard.Close() // Done with control channel

			fmt.Println("等待 10 秒...")
			time.Sleep(10 * time.Second)

			// 3. Connect Terminal
			fmt.Println("[3/4] 连接设备终端...")

			// Use Seat as DeviceID for terminal connection
			targetID := int64(target.Seat)
			wsTerm, err := connService.ConnectDeviceTerminal(serverConfig, targetID)
			if err != nil {
				return fmt.Errorf("连接终端失败: %v", err)
			}

			term := terminal.NewTerminalSession(wsTerm, targetID)
			defer term.Close()

			if err := term.Init(); err != nil {
				return fmt.Errorf("终端初始化失败: %v", err)
			}

			fmt.Println("等待终端就绪 (等待 '$' 提示符)...")

			timeout := time.After(5 * time.Second)

		WaitLoop:
			for {
				select {
				case line := <-term.Output:
					if strings.Contains(line, "$") || strings.Contains(line, "#") {
						break WaitLoop
					}
				case <-timeout:
					return fmt.Errorf("等待终端就绪超时 (5秒)")
				case <-term.Closed:
					return fmt.Errorf("终端连接意外关闭")
				}
			}

			// 4. Execute Command
			cmdStr := "tail -100 /data/local/tmp/guard/logs/guard.log"
			fmt.Printf("[4/4] 执行命令: %s\n", cmdStr)
			fmt.Println("------------------------------------------------")

			if err := term.Exec(cmdStr); err != nil {
				return fmt.Errorf("发送命令失败: %v", err)
			}

			// 5. Stream Output
			go func() {
				for line := range term.Output {
					fmt.Print(line)
				}
			}()

			// Block until signal
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			<-sigChan

			fmt.Println("\n\n正在清理资源...")

			// Cleanup 1: Disable ADB via Terminal
			fmt.Println("[1/2] 关闭 ADB...")
			if err := term.Exec("settings put global adb_enabled 0"); err != nil {
				fmt.Printf("关闭 ADB 失败: %v\n", err)
			} else {
				// Give it a moment to process
				time.Sleep(1 * time.Second)
			}
			term.Close()

			// Cleanup 2: Switch to OTG via Guard
			fmt.Println("[2/2] 切换到 OTG 模式...")
			wsGuardCleanup, err := connService.ConnectGuard(serverConfig)
			if err != nil {
				return fmt.Errorf("连接 Guard 失败: %v", err)
			}
			defer wsGuardCleanup.Close()

			if err := api_SwitchUSBMode(wsGuardCleanup, target.Seat, true); err != nil {
				return fmt.Errorf("切换 OTG 模式失败: %v", err)
			}
			fmt.Println("清理完成。")

			return nil
		},
	}

	cmd.Flags().StringVarP(&groupFilter, "group", "g", "", "指定分组")
	cmd.Flags().StringVarP(&serverFilter, "server", "s", "", "指定服务器 (IP/URL)")
	cmd.Flags().StringVarP(&uuidFilter, "uuid", "u", "", "指定设备 UUID")
	cmd.Flags().IntVar(&seatFilter, "seat", -1, "指定机位号")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "交互模式")
	cmd.Flags().StringVar(&authorizedOnly, "authorized", "", "筛选授权状态 (true/false)")
	// 允许 --authorized 等同于 --authorized=true
	cmd.Flags().Lookup("authorized").NoOptDefVal = "true"

	return cmd
}

func api_SwitchUSBMode(client interface{}, seat int, otg bool) error {
	c, ok := client.(*wsclient.Client)
	if !ok {
		return fmt.Errorf("invalid client")
	}

	mode := 1
	if otg {
		mode = 0
	}

	_, err := c.SendRequest(model.FuncSwitchUSBGuard, map[string]interface{}{
		"seat": seat,
		"mode": mode,
	})
	return err
}

func api_ControlADB(client interface{}, seat int, enable bool) error {
	c, ok := client.(*wsclient.Client)
	if !ok {
		return fmt.Errorf("invalid client")
	}

	mode := 0
	if enable {
		mode = 2
	}

	_, err := c.SendRequest(model.FuncEnableADB, map[string]interface{}{
		"seat": seat,
		"mode": mode,
	})
	return err
}

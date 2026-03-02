package cloud

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	cloudPkg "jpy-cli/pkg/cloud"
	"jpy-cli/pkg/cloud/stress"
	"jpy-cli/pkg/tui"

	"github.com/spf13/cobra"
)

// NewStressCmd 创建改机压力测试命令
func NewStressCmd() *cobra.Command {
	var (
		all        bool
		loop       int
		interval   time.Duration
		timeout    time.Duration
		configPath string
		output     string // plain / json / tui
	)

	cmd := &cobra.Command{
		Use:   "stress",
		Short: "改机压力测试",
		Long: `对集控平台设备执行改机压力测试。

输出模式:
  --output tui     交互式 TUI 界面（默认）
  --output plain   纯文本输出，适合 SSH / 脚本环境
  --output json    JSON 格式输出，适合程序对接

示例:
  jpy cloud stress --all --config configs/zh.json --output plain
  jpy cloud stress --all --loop 5 --interval 3m --timeout 10m
  jpy cloud stress --config configs/us.json --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. 确保已连接
			if err := cloudPkg.EnsureConnected(); err != nil {
				return err
			}

			// 2. 获取设备列表
			var devices []stress.DeviceInfo
			var err error

			if all {
				// 全量模式
				fmt.Println("正在获取所有设备...")
				devices, err = stress.GetAllDevices()
				if err != nil {
					return err
				}
				fmt.Printf("共发现 %d 台设备\n", len(devices))
			} else if output == "plain" || output == "json" {
				// 非交互模式下 --all 是必须的
				return fmt.Errorf("非交互模式 (--output %s) 必须指定 --all", output)
			} else {
				// TUI 交互选择
				devices, err = selectDevicesTUI()
				if err != nil {
					return err
				}
			}

			if len(devices) == 0 {
				fmt.Println("未找到设备。")
				return nil
			}

			// 3. 确定改机配置文件
			if configPath == "" {
				cfg, _ := cloudPkg.LoadCloudConfig()
				if cfg != nil && cfg.LastUsedConfig != "" {
					configPath = cfg.LastUsedConfig
				}
			}
			if configPath == "" {
				if output == "plain" || output == "json" {
					return fmt.Errorf("非交互模式必须指定 --config 参数")
				}
				// TUI 选择配置文件
				selected, err := selectConfigTUI()
				if err != nil {
					return err
				}
				configPath = selected
			}

			// 验证配置文件存在
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				return fmt.Errorf("改机配置文件不存在: %s", configPath)
			}

			// 记住上次使用的配置
			if cloudCfg, err := cloudPkg.LoadCloudConfig(); err == nil {
				cloudCfg.LastUsedConfig = configPath
				_ = cloudPkg.SaveCloudConfig(cloudCfg)
			}

			fmt.Printf("使用配置: %s\n", configPath)
			fmt.Printf("设备数量: %d | 循环: %d | 间隔: %v | 超时: %v\n",
				len(devices), loop, interval, timeout)

			// 4. 创建并运行压力测试
			// 读取通知配置
			cloudCfg2, _ := cloudPkg.LoadCloudConfig()
			logDir := ""
			if cloudCfg2 != nil {
				logDir = cloudPkg.GetCloudConfigDir() + "/logs"
			}

			cfg := stress.StressConfig{
				ConfigPath:              configPath,
				Loop:                    loop,
				Interval:                interval,
				Timeout:                 timeout,
				LogDir:                  logDir,
			}
			if cloudCfg2 != nil {
				cfg.LarkWebhook = cloudCfg2.Notification.LarkWebhook
				cfg.NotifyEnabled = cloudCfg2.Notification.Enabled
				cfg.NotifyAlways = cloudCfg2.Notification.NotifyAlways
				cfg.OfflineThresholdPercent = cloudCfg2.Notification.OfflineThresholdPercent
			}

			switch output {
			case "json":
				return runStressJSON(cfg, devices)
			case "plain":
				return runStressPlain(cfg, devices)
			default:
				return runStressTUI(cfg, devices)
			}
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "全量模式：测试所有设备")
	cmd.Flags().IntVar(&loop, "loop", 1, "循环次数 (0=无限)")
	cmd.Flags().DurationVar(&interval, "interval", 3*time.Minute, "轮次间隔")
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Minute, "单轮超时")
	cmd.Flags().StringVar(&configPath, "config", "", "改机参数配置文件路径 (JSON)")
	cmd.Flags().StringVarP(&output, "output", "o", "tui", "输出模式 (tui/plain/json)")

	return cmd
}

// runStressPlain 纯文本模式运行压力测试
func runStressPlain(cfg stress.StressConfig, devices []stress.DeviceInfo) error {
	runner := stress.NewRunner(cfg, devices, func(event string, data interface{}) {
		switch event {
		case "round_start":
			d := data.(map[string]interface{})
			fmt.Printf("\n[轮次 %v] 开始，设备数: %v\n", d["round"], d["total"])
		case "online_check_done":
			d := data.(map[string]interface{})
			fmt.Printf("[在线检查] 总数: %v, 在线: %v\n", d["total"], d["online"])
		case "batch_send":
			d := data.(map[string]interface{})
			fmt.Printf("[发送] %v-%v / %v\n", d["from"], d["to"], d["total"])
		case "all_sent":
			d := data.(map[string]interface{})
			fmt.Printf("[发送完毕] 共 %v 个任务\n", d["task_count"])
		case "poll_update":
			d := data.(map[string]interface{})
			fmt.Printf("\r[轮询] 成功: %v | 失败: %v | 等待: %v / %v",
				d["success"], d["fail"], d["pending"], d["total"])
		case "round_done":
			s := data.(stress.RoundSummary)
			fmt.Printf("\n[完成] 轮次 %d - 成功: %d | 失败: %d | 超时: %d | 耗时: %v\n",
				s.Round, s.SuccessCount, s.FailCount, s.TimeoutCount, s.Duration.Round(time.Second))
			if len(s.FailReasons) > 0 {
				fmt.Println("  失败原因:")
				for i, r := range s.FailReasons {
					if i >= 5 {
						break
					}
					fmt.Printf("    %d. %s (%d 次)\n", i+1, r.Reason, r.Count)
				}
			}
		case "interval_wait":
			d := data.(map[string]interface{})
			fmt.Printf("[等待] 轮次 %v 结束，等待 %v 开始下一轮...\n", d["round"], d["interval"])
		}
	})

	runner.SetLogger(func(level, format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		fmt.Fprintf(os.Stderr, "[%s] [%s] %s\n", time.Now().Format("15:04:05"), level, msg)
	})

	summaries, err := runner.Run()
	if err != nil {
		return err
	}

	fmt.Println("\n========== 总结 ==========")
	fmt.Print(stress.FormatSummaryPlain(summaries))

	return nil
}

// runStressJSON JSON 模式运行压力测试
func runStressJSON(cfg stress.StressConfig, devices []stress.DeviceInfo) error {
	runner := stress.NewRunner(cfg, devices, nil)

	summaries, err := runner.Run()
	if err != nil {
		return err
	}

	output, err := stress.FormatSummaryJSON(summaries)
	if err != nil {
		return err
	}

	fmt.Println(output)
	return nil
}

// runStressTUI TUI 模式运行压力测试（使用回调式进度显示，不使用 BubbleTea raw mode）
func runStressTUI(cfg stress.StressConfig, devices []stress.DeviceInfo) error {
	runner := stress.NewRunner(cfg, devices, func(event string, data interface{}) {
		switch event {
		case "round_start":
			d := data.(map[string]interface{})
			fmt.Printf("\n>>> 轮次 %v 开始，设备数: %v\n", d["round"], d["total"])
		case "online_check_done":
			d := data.(map[string]interface{})
			fmt.Printf("    在线检查完成: %v/%v 台在线\n", d["online"], d["total"])
		case "batch_send":
			d := data.(map[string]interface{})
			fmt.Printf("    发送改机请求: %v-%v / %v\n", d["from"], d["to"], d["total"])
		case "all_sent":
			d := data.(map[string]interface{})
			fmt.Printf("    请求已全部发送，共 %v 个任务\n", d["task_count"])
		case "poll_update":
			d := data.(map[string]interface{})
			fmt.Printf("\r    进度: 成功 %v | 失败 %v | 等待 %v / %v",
				d["success"], d["fail"], d["pending"], d["total"])
		case "round_done":
			s := data.(stress.RoundSummary)
			fmt.Printf("\n    轮次 %d 完成 - 成功: %d | 失败: %d | 超时: %d | 耗时: %v\n",
				s.Round, s.SuccessCount, s.FailCount, s.TimeoutCount, s.Duration.Round(time.Second))
		case "interval_wait":
			d := data.(map[string]interface{})
			fmt.Printf("    等待 %v 开始下一轮...\n", d["interval"])
		}
	})

	summaries, err := runner.Run()
	if err != nil {
		return err
	}

	fmt.Println("\n========== 总结 ==========")
	fmt.Print(stress.FormatSummaryPlain(summaries))

	return nil
}

// selectDevicesTUI 交互式选择设备（TUI 多选）
func selectDevicesTUI() ([]stress.DeviceInfo, error) {
	fmt.Println("正在获取设备列表...")
	allDevices, err := stress.GetAllDevices()
	if err != nil {
		return nil, err
	}

	if len(allDevices) == 0 {
		return nil, fmt.Errorf("未找到设备")
	}

	// 构建选项列表
	options := make([]tui.Option, len(allDevices))
	for i, d := range allDevices {
		status := "在线"
		if !d.Online {
			status = "离线"
		}
		options[i] = tui.Option{
			Label: fmt.Sprintf("%d | %s | %s | %s [%s]", d.DeviceID, d.UUID, d.Brand, d.IP, status),
			Value: strconv.FormatInt(d.DeviceID, 10),
		}
	}

	selected, err := tui.SelectOption("请选择设备 (Enter 确认):", "", options)
	if err != nil {
		return nil, err
	}

	// 根据选中的 ID 过滤设备
	for _, d := range allDevices {
		if strconv.FormatInt(d.DeviceID, 10) == selected {
			return []stress.DeviceInfo{d}, nil
		}
	}

	return allDevices, nil
}

// selectConfigTUI 交互式选择改机配置文件
func selectConfigTUI() (string, error) {
	dir := cloudPkg.GetChangeOsConfigsDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return "", fmt.Errorf("改机配置目录不存在: %s\n请先运行: jpy cloud config init-configs", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("读取配置目录失败: %w", err)
	}

	var options []tui.Option
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			options = append(options, tui.Option{
				Label: e.Name(),
				Value: fmt.Sprintf("%s/%s", dir, e.Name()),
			})
		}
	}

	if len(options) == 0 {
		return "", fmt.Errorf("未找到配置文件。请先运行: jpy cloud config init-configs")
	}

	selected, err := tui.SelectOption("请选择改机配置文件:", "", options)
	if err != nil {
		return "", err
	}

	return selected, nil
}

// FormatDeviceListJSON 格式化设备列表为 JSON
func FormatDeviceListJSON(devices []stress.DeviceInfo) string {
	data, _ := json.MarshalIndent(devices, "", "  ")
	return string(data)
}

package stress

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"adminApi"

	"cnb.cool/accbot/goTool/sessionPkg"
	"github.com/spf13/cobra"
)

// 全局 SDK 客户端
var (
	sdkClient  *adminApi.AdminApi
	sdkSession *sessionPkg.Session
)

// StressLogger 压力测试专用日志器
type StressLogger struct {
	file   *os.File
	mu     sync.Mutex
	stdout bool // 是否同时输出到终端
}

func newStressLogger(logPath string, stdout bool) (*StressLogger, error) {
	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %v", err)
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("创建日志文件失败: %v", err)
	}

	return &StressLogger{file: f, stdout: stdout}, nil
}

func (l *StressLogger) Log(level, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("[%s] [%s] %s\n", timestamp, level, msg)

	l.file.WriteString(logLine)

	if l.stdout {
		fmt.Print(logLine)
	}
}

func (l *StressLogger) Info(format string, args ...interface{}) {
	l.Log("INFO", format, args...)
}

func (l *StressLogger) Error(format string, args ...interface{}) {
	l.Log("ERROR", format, args...)
}

func (l *StressLogger) Warn(format string, args ...interface{}) {
	l.Log("WARN", format, args...)
}

func (l *StressLogger) Close() {
	if l.file != nil {
		l.file.Close()
	}
}

// ChangeOsParams 改机参数
type ChangeOsParams struct {
	Bs           string `json:"bs"`
	Category     string `json:"category"`
	Version      string `json:"version"`
	Country      string `json:"country"`
	Language     string `json:"language"`
	Timezone     string `json:"timezone"`
	OperatorName string `json:"operatorName"`
	Mcc          string `json:"mcc"`
	Mnc          string `json:"mnc"`
	Operator     string `json:"operator"`
	Msisdn       string `json:"msisdn"`
	Smsc         string `json:"smsc"`
}

// DeviceResult 设备测试结果
type DeviceResult struct {
	DeviceID int64  `json:"device_id"`
	UUID     string `json:"uuid"`
	IP       string `json:"ip"`
	Success  bool   `json:"success"`
	Error    string `json:"error,omitempty"`
	Duration string `json:"duration,omitempty"`
}

// RoundResult 单轮测试结果
type RoundResult struct {
	Round    int            `json:"round"`
	Total    int            `json:"total"`
	Success  int            `json:"success"`
	Failed   int            `json:"failed"`
	Skipped  int            `json:"skipped"`
	Duration string         `json:"duration"`
	Devices  []DeviceResult `json:"devices,omitempty"`
}

func newUserCmd() *cobra.Command {
	var (
		serverURL  string
		secretKey  string
		configFile string
		loop       int
		interval   time.Duration
		timeout    time.Duration
		deviceIDs  []int64
		output     string
		logDir     string
		debug      bool
	)

	cmd := &cobra.Command{
		Use:   "user",
		Short: "用户端改机压力测试",
		Long: `用户端改机压力测试，支持一行命令执行。

示例:
  # 测试所有设备，单次
  jpy stress user -s wss://home.accjs.cn/ws -k YOUR_SECRET_KEY -c config.json

  # 测试指定设备，循环3次，间隔5分钟
  jpy stress user -s wss://home.accjs.cn/ws -k YOUR_SECRET_KEY -c config.json --device 123,456 --loop 3 --interval 5m

  # 无限循环测试
  jpy stress user -s wss://home.accjs.cn/ws -k YOUR_SECRET_KEY -c config.json --loop 0 --interval 3m

  # 调试模式：遇到失败立即停止（配合 --loop 0 保留现场）
  jpy stress user -s wss://home.accjs.cn/ws -k YOUR_SECRET_KEY -c config.json --loop 0 --debug`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStressUser(serverURL, secretKey, configFile, loop, interval, timeout, deviceIDs, output, logDir, debug)
		},
	}

	cmd.Flags().StringVarP(&serverURL, "server", "s", "wss://home.accjs.cn/ws", "WebSocket 服务地址")
	cmd.Flags().StringVarP(&secretKey, "key", "k", "", "登录密钥（必填）")
	cmd.Flags().StringVarP(&configFile, "config", "c", "", "改机配置文件路径（必填）")
	cmd.Flags().Int64SliceVar(&deviceIDs, "device", nil, "指定设备ID列表（逗号分隔），不指定则测试所有设备")
	cmd.Flags().IntVar(&loop, "loop", 1, "循环次数（0=无限循环）")
	cmd.Flags().DurationVar(&interval, "interval", 3*time.Minute, "循环间隔时间")
	cmd.Flags().DurationVar(&timeout, "timeout", 10*time.Minute, "单轮超时时间")
	cmd.Flags().StringVarP(&output, "output", "o", "plain", "输出模式: plain/json")
	cmd.Flags().StringVar(&logDir, "log-dir", "", "日志目录（默认 ~/.jpy/logs/stress）")
	cmd.Flags().BoolVar(&debug, "debug", false, "调试模式：遇到失败立即停止（配合 --loop 0 保留现场）")

	cmd.MarkFlagRequired("key")
	cmd.MarkFlagRequired("config")

	return cmd
}

func runStressUser(serverURL, secretKey, configFile string, loop int, interval, timeout time.Duration, deviceIDs []int64, output, logDir string, debug bool) error {
	// 1. 初始化日志
	if logDir == "" {
		home, _ := os.UserHomeDir()
		logDir = filepath.Join(home, ".jpy", "logs", "stress")
	}
	logFile := filepath.Join(logDir, fmt.Sprintf("stress_user_%s.log", time.Now().Format("20060102_150405")))

	logger, err := newStressLogger(logFile, true)
	if err != nil {
		return err
	}
	defer logger.Close()

	taskBatchID := fmt.Sprintf("STRESS-USER-%s", time.Now().Format("20060102150405"))
	logger.Info("========================================")
	logger.Info("任务编号: %s", taskBatchID)
	logger.Info("服务地址: %s", serverURL)
	logger.Info("配置文件: %s", configFile)
	logger.Info("循环次数: %d (0=无限)", loop)
	logger.Info("循环间隔: %v", interval)
	logger.Info("单轮超时: %v", timeout)
	if debug {
		logger.Info("调试模式: 开启（遇到失败立即停止）")
	}
	logger.Info("日志文件: %s", logFile)
	logger.Info("========================================")

	// 2. 读取改机配置
	configContent, err := os.ReadFile(configFile)
	if err != nil {
		logger.Error("读取配置文件失败: %v", err)
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	var params ChangeOsParams
	if err := json.Unmarshal(configContent, &params); err != nil {
		logger.Error("解析配置文件失败: %v", err)
		return fmt.Errorf("解析配置文件失败: %v", err)
	}
	logger.Info("配置解析成功: bs=%s, category=%s", params.Bs, params.Category)

	// 3. 初始化 SDK（静默 SDK 内部日志）
	if err := initSDKSilent(serverURL); err != nil {
		logger.Error("SDK 初始化失败: %v", err)
		return fmt.Errorf("SDK 初始化失败: %v", err)
	}
	logger.Info("SDK 连接成功")

	// 4. 登录
	if err := loginWithSecret(secretKey); err != nil {
		logger.Error("登录失败: %v", err)
		return fmt.Errorf("登录失败: %v", err)
	}
	logger.Info("登录成功")

	// 5. 获取设备列表
	var targetIDs []int64
	deviceInfoMap := make(map[int64]struct {
		UUID string
		IP   string
	})

	if len(deviceIDs) > 0 {
		targetIDs = deviceIDs
		logger.Info("使用指定设备: %v", deviceIDs)
	} else {
		logger.Info("正在获取所有设备...")
		ids, infoMap, err := fetchAllDevices(logger)
		if err != nil {
			return err
		}
		targetIDs = ids
		deviceInfoMap = infoMap
		logger.Info("共获取 %d 台设备", len(targetIDs))
	}

	if len(targetIDs) == 0 {
		logger.Warn("未找到设备，退出")
		return fmt.Errorf("未找到设备")
	}

	// 6. 开始循环测试
	var allResults []RoundResult
	round := 1

	for {
		logger.Info("========== 第 %d 轮开始 ==========", round)
		roundStart := time.Now()

		// 过滤在线设备
		onlineIDs := filterOnlineDevices(targetIDs, logger)
		logger.Info("在线设备: %d / %d", len(onlineIDs), len(targetIDs))

		var roundResult RoundResult
		roundResult.Round = round
		roundResult.Total = len(targetIDs)
		roundResult.Skipped = len(targetIDs) - len(onlineIDs)

		if len(onlineIDs) == 0 {
			logger.Warn("无在线设备，跳过本轮")
		} else {
			// 执行改机
			success, failed, deviceResults := performChangeOs(onlineIDs, &params, logger, timeout, deviceInfoMap, round, debug)
			roundResult.Success = success
			roundResult.Failed = failed
			roundResult.Devices = deviceResults

			// 调试模式：遇到失败立即停止
			if debug && failed > 0 {
				logger.Error("调试模式：检测到 %d 台设备失败，立即停止以保留现场", failed)
				roundResult.Duration = time.Since(roundStart).Round(time.Second).String()
				allResults = append(allResults, roundResult)
				break
			}
		}

		roundResult.Duration = time.Since(roundStart).Round(time.Second).String()
		allResults = append(allResults, roundResult)

		logger.Info("第 %d 轮完成: 成功=%d, 失败=%d, 跳过=%d, 耗时=%s",
			round, roundResult.Success, roundResult.Failed, roundResult.Skipped, roundResult.Duration)

		// 检查是否继续
		if loop > 0 && round >= loop {
			logger.Info("达到循环上限 (%d)，退出", loop)
			break
		}

		logger.Info("等待 %v 后开始下一轮...", interval)
		time.Sleep(interval)
		round++
	}

	// 7. 输出最终结果
	logger.Info("========================================")
	logger.Info("测试完成，共 %d 轮", len(allResults))

	if output == "json" {
		data, _ := json.MarshalIndent(allResults, "", "  ")
		fmt.Println(string(data))
	} else {
		fmt.Printf("\n压力测试完成\n")
		fmt.Printf("任务编号: %s\n", taskBatchID)
		fmt.Printf("总轮次: %d\n", len(allResults))
		fmt.Printf("日志文件: %s\n", logFile)

		totalSuccess, totalFailed := 0, 0
		for _, r := range allResults {
			totalSuccess += r.Success
			totalFailed += r.Failed
		}
		fmt.Printf("累计成功: %d, 累计失败: %d\n", totalSuccess, totalFailed)
	}

	return nil
}

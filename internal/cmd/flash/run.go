package flash

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// FlashResult 刷机结果
type FlashResult struct {
	COM      string        `json:"com"`
	Channel  int           `json:"channel"`
	IP       string        `json:"ip"`
	UUID     string        `json:"uuid,omitempty"`
	Success  bool          `json:"success"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration"`
}

// FlashTask 刷机任务
type FlashTask struct {
	COM     string
	Channel int
	IP      string
}

// DeviceInfo 设备信息
type DeviceInfo struct {
	Online bool   `json:"online"`
	UUID   string `json:"uuid"`
	IP     string `json:"ip"`
	Seat   int    `json:"seat"`
}

// DeviceListResp 设备列表响应
type DeviceListResp struct {
	Total   int          `json:"total"`
	Devices []DeviceInfo `json:"devices"`
}

var (
	middleware  string
	user        string
	pass        string
	flashScript string
	comPort     string
	channels    string
	ipPrefix    string // IP 前缀，如 172.25.0 或 192.168.11
	ipOffset    int    // IP 偏移量，设备IP = ip-prefix.(ip-offset + 通道号)
	dryRun      bool
	timeout     int
	jpyPath     string
	skipOffline bool
	retryCount  int
	output      string
	autoConfirm bool
	remoteAddr  string // 远程 jpy server 地址
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "执行批量刷机",
		Long: `执行批量刷机任务。

IP 计算规则:
  设备IP = {ip-prefix}.{ip-offset + 通道号 - 1}
  即 ip-offset 是通道1的起始IP
  例: --ip-prefix 172.25.0 --ip-offset 11 --ch 1 => 172.25.0.11
      --ip-prefix 172.25.0 --ip-offset 11 --ch 3 => 172.25.0.13

示例:
  # 刷 COM3 通道1（IP: 172.25.0.11）
  jpy flash run --com COM3 --ch 1 --mw 172.25.0.251 --ip-prefix 172.25.0 --ip-offset 11 --script D:\flash\flash.cmd

  # 刷 COM3 的 1-10 通道（IP: 172.25.0.11-20）
  jpy flash run --com COM3 --ch 1-10 --mw 172.25.0.251 --ip-prefix 172.25.0 --ip-offset 11 --script D:\flash\flash.cmd

  # 刷指定通道 1,3,5（IP: 172.25.0.11, 13, 15）
  jpy flash run --com COM3 --ch 1,3,5 --mw 172.25.0.251 --ip-prefix 172.25.0 --ip-offset 11 --script D:\flash\flash.cmd

  # 模拟运行（查看 IP 映射）
  jpy flash run --com COM3 --ch 1-5 --mw 172.25.0.251 --ip-prefix 172.25.0 --ip-offset 11 --script D:\flash\flash.cmd --dry

  # 远程执行（COM口在远程机器上）
  jpy flash run --remote 192.168.1.100:9090 --com COM3 --ch 1 --mw 172.25.0.251 --ip-prefix 172.25.0 --ip-offset 11 --script D:\flash\flash.cmd`,
		RunE: runFlash,
	}

	cmd.Flags().StringVar(&middleware, "mw", "", "中间件地址（必填）")
	cmd.Flags().StringVarP(&user, "user", "u", "admin", "中间件用户名")
	cmd.Flags().StringVarP(&pass, "pass", "p", "admin", "中间件密码")
	cmd.Flags().StringVar(&flashScript, "script", "", "刷机脚本路径（必填）")
	cmd.Flags().StringVar(&comPort, "com", "", "COM口: COM3, COM4, COM6 或 all（必填）")
	cmd.Flags().StringVar(&channels, "ch", "all", "通道: 1,2,3 或 1-20 或 all")
	cmd.Flags().StringVar(&ipPrefix, "ip-prefix", "", "IP 前缀（必填，如 172.25.0 或 192.168.11）")
	cmd.Flags().IntVar(&ipOffset, "ip-offset", -1, "通道1的起始IP（必填，通道N的IP = ip-offset + N - 1）")
	cmd.Flags().BoolVar(&dryRun, "dry", false, "模拟运行")
	cmd.Flags().IntVar(&timeout, "timeout", 600, "单台刷机超时(秒)")
	cmd.Flags().StringVar(&jpyPath, "jpy", "jpy", "jpy工具路径")
	cmd.Flags().BoolVar(&skipOffline, "skip-offline", true, "跳过离线设备")
	cmd.Flags().IntVar(&retryCount, "retry", 1, "失败重试次数")
	cmd.Flags().StringVarP(&output, "output", "o", "plain", "输出模式: plain/json")
	cmd.Flags().BoolVarP(&autoConfirm, "yes", "y", false, "跳过确认直接执行")
	cmd.Flags().StringVar(&remoteAddr, "remote", "", "远程 jpy server 地址（如 192.168.1.100:9090）")

	cmd.MarkFlagRequired("mw")
	cmd.MarkFlagRequired("script")
	cmd.MarkFlagRequired("com")
	cmd.MarkFlagRequired("ip-prefix")
	cmd.MarkFlagRequired("ip-offset")

	return cmd
}

func runFlash(cmd *cobra.Command, args []string) error {
	printBanner()

	// 解析任务
	tasks := parseTasks()
	if len(tasks) == 0 {
		return fmt.Errorf("没有找到要刷的设备")
	}

	// 显示任务列表
	logInfo("", 0, "共 %d 台设备待刷机", len(tasks))
	for _, t := range tasks {
		logInfo(t.COM, t.Channel, "%s", t.IP)
	}

	// 确认
	if !autoConfirm && !dryRun {
		if !confirm("确认开始刷机?") {
			logInfo("", 0, "已取消")
			return nil
		}
	}

	// 开始刷机
	results := make([]FlashResult, 0)
	startTime := time.Now()

	for i, task := range tasks {
		logInfo(task.COM, task.Channel, "========== [%d/%d] 开始 ==========", i+1, len(tasks))

		result := flashDeviceWithRetry(task, retryCount)
		results = append(results, result)

		if result.Success {
			logInfo(task.COM, task.Channel, "SUCCESS 耗时 %.1f秒", result.Duration.Seconds())
		} else {
			logError(task.COM, task.Channel, "FAILED: %s", result.Error)
		}
	}

	// 汇总
	printSummary(results, time.Since(startTime))

	// JSON 输出
	if output == "json" {
		data, _ := json.Marshal(map[string]interface{}{
			"results":    results,
			"total_time": time.Since(startTime).String(),
		})
		fmt.Println(string(data))
	}

	return nil
}

func parseTasks() []FlashTask {
	tasks := make([]FlashTask, 0)

	// 解析 COM 口
	coms := []string{}
	if strings.ToLower(comPort) == "all" {
		coms = []string{"COM3", "COM4", "COM6"}
	} else {
		for _, c := range strings.Split(comPort, ",") {
			coms = append(coms, strings.TrimSpace(strings.ToUpper(c)))
		}
	}

	// 解析通道
	chList := parseChannels(channels)

	for _, c := range coms {
		for _, ch := range chList {
			if ch < 1 || ch > 20 {
				continue
			}
			// IP = ip-prefix.(ip-offset + 通道号 - 1)
			// 即 ip-offset 是通道1的起始IP
			// 例: --ip-offset 11 --ch 1 => 11, --ch 3 => 13
			tasks = append(tasks, FlashTask{
				COM:     c,
				Channel: ch,
				IP:      fmt.Sprintf("%s.%d", ipPrefix, ipOffset+ch-1),
			})
		}
	}

	return tasks
}

func parseChannels(ch string) []int {
	if ch == "" || strings.ToLower(ch) == "all" {
		result := make([]int, 20)
		for i := 0; i < 20; i++ {
			result[i] = i + 1
		}
		return result
	}

	result := make([]int, 0)
	for _, p := range strings.Split(ch, ",") {
		p = strings.TrimSpace(p)
		if strings.Contains(p, "-") {
			parts := strings.Split(p, "-")
			if len(parts) == 2 {
				start, _ := strconv.Atoi(parts[0])
				end, _ := strconv.Atoi(parts[1])
				for i := start; i <= end; i++ {
					result = append(result, i)
				}
			}
		} else {
			if n, err := strconv.Atoi(p); err == nil && n > 0 {
				result = append(result, n)
			}
		}
	}
	return result
}

func flashDeviceWithRetry(task FlashTask, maxRetry int) FlashResult {
	var result FlashResult
	for i := 0; i < maxRetry; i++ {
		if i > 0 {
			logInfo(task.COM, task.Channel, "重试 %d/%d", i, maxRetry-1)
			time.Sleep(5 * time.Second)
		}
		result = flashDevice(task)
		if result.Success || result.Error == "设备离线" {
			break
		}
	}
	return result
}

func flashDevice(task FlashTask) FlashResult {
	result := FlashResult{
		COM:     task.COM,
		Channel: task.Channel,
		IP:      task.IP,
	}
	startTime := time.Now()

	if dryRun {
		logInfo(task.COM, task.Channel, "[模拟] 检查设备状态")
		logInfo(task.COM, task.Channel, "[模拟] reboot bootloader")
		logInfo(task.COM, task.Channel, "[模拟] 切换 HUB 模式")
		logInfo(task.COM, task.Channel, "[模拟] 执行刷机脚本")
		logInfo(task.COM, task.Channel, "[模拟] 切换 OTG 模式")
		time.Sleep(2 * time.Second)
		result.Success = true
		result.Duration = time.Since(startTime)
		return result
	}

	// Step 1: 检查设备状态
	logInfo(task.COM, task.Channel, "[1/5] 检查设备状态...")
	online, uuid := checkDevice(task.IP)
	if !online {
		if skipOffline {
			result.Error = "设备离线"
			result.Duration = time.Since(startTime)
			return result
		}
	}
	result.UUID = uuid
	if uuid != "" {
		logInfo(task.COM, task.Channel, "设备在线 UUID=%s", uuid)
	}

	// Step 2: 发送 reboot bootloader
	logInfo(task.COM, task.Channel, "[2/5] 发送 reboot bootloader...")
	if err := rebootBootloader(task.IP); err != nil {
		result.Error = fmt.Sprintf("reboot bootloader 失败: %v", err)
		result.Duration = time.Since(startTime)
		return result
	}
	time.Sleep(5 * time.Second)

	// Step 3: 切换 HUB 模式
	logInfo(task.COM, task.Channel, "[3/5] 切换为 HUB 模式...")
	if err := setMode(task.COM, task.Channel, "hub"); err != nil {
		result.Error = fmt.Sprintf("切换HUB失败: %v", err)
		result.Duration = time.Since(startTime)
		return result
	}
	time.Sleep(10 * time.Second)

	// Step 4: 执行刷机脚本
	logInfo(task.COM, task.Channel, "[4/5] 执行刷机脚本 (超时%d秒)...", timeout)
	if err := runFlashScript(); err != nil {
		// 失败也要切回 OTG
		setMode(task.COM, task.Channel, "otg")
		result.Error = fmt.Sprintf("刷机失败: %v", err)
		result.Duration = time.Since(startTime)
		return result
	}

	// Step 5: 切换 OTG 模式
	logInfo(task.COM, task.Channel, "[5/5] 切换为 OTG 模式...")
	if err := setMode(task.COM, task.Channel, "otg"); err != nil {
		result.Error = fmt.Sprintf("切换OTG失败: %v", err)
		result.Duration = time.Since(startTime)
		return result
	}

	result.Success = true
	result.Duration = time.Since(startTime)
	return result
}

func checkDevice(ip string) (bool, string) {
	cmd := exec.Command(jpyPath, "device", "list",
		"-s", middleware, "-u", user, "-p", pass,
		"--ip", ip, "-o", "json")

	output, err := cmd.Output()
	if err != nil {
		return false, ""
	}

	var resp DeviceListResp
	if err := json.Unmarshal(output, &resp); err != nil {
		return false, ""
	}

	if resp.Total > 0 && len(resp.Devices) > 0 {
		d := resp.Devices[0]
		return d.Online, d.UUID
	}
	return false, ""
}

func rebootBootloader(ip string) error {
	cmd := exec.Command(jpyPath, "device", "shell", "reboot bootloader",
		"-s", middleware, "-u", user, "-p", pass,
		"--ip", ip)
	return cmd.Run()
}

func setMode(port string, channel int, mode string) error {
	var cmd *exec.Cmd
	if remoteAddr != "" {
		// 远程模式：通过 --remote 转发 COM 操作
		cmd = exec.Command(jpyPath, "--remote", remoteAddr, "com", "set-mode",
			"--port", port, "--mode", mode, "--channel", strconv.Itoa(channel))
	} else {
		// 本地模式
		cmd = exec.Command(jpyPath, "com", "set-mode",
			"--port", port, "--mode", mode, "--channel", strconv.Itoa(channel))
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, string(output))
	}
	return nil
}

func runFlashScript() error {
	// 跨平台处理
	var cmd *exec.Cmd
	if strings.Contains(flashScript, "\\") {
		// Windows 路径
		dir := flashScript[:strings.LastIndex(flashScript, "\\")]
		script := flashScript[strings.LastIndex(flashScript, "\\")+1:]
		cmd = exec.Command("cmd", "/c", "cd", "/d", dir, "&&", script)
	} else {
		// Unix 路径
		cmd = exec.Command("sh", "-c", flashScript)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(time.Duration(timeout) * time.Second):
		cmd.Process.Kill()
		return fmt.Errorf("超时 (%d秒)", timeout)
	}
}

func printBanner() {
	fmt.Fprintln(os.Stderr, "╔════════════════════════════════════════════════╗")
	fmt.Fprintln(os.Stderr, "║           JPY 批量刷机工具 v1.0                ║")
	fmt.Fprintln(os.Stderr, "╚════════════════════════════════════════════════╝")
	fmt.Fprintf(os.Stderr, "中间件: %s (用户: %s)\n", middleware, user)
	fmt.Fprintf(os.Stderr, "刷机脚本: %s\n", flashScript)
	fmt.Fprintf(os.Stderr, "COM口: %s | 通道: %s | IP起始: %s.%d\n", comPort, channels, ipPrefix, ipOffset)
	if remoteAddr != "" {
		fmt.Fprintf(os.Stderr, "远程模式: %s\n", remoteAddr)
	}
	fmt.Fprintf(os.Stderr, "超时: %d秒 | 重试: %d次\n", timeout, retryCount)
	if dryRun {
		fmt.Fprintln(os.Stderr, "*** 模拟运行模式 ***")
	}
}

func printSummary(results []FlashResult, totalTime time.Duration) {
	fmt.Fprintln(os.Stderr, "\n"+strings.Repeat("=", 50))
	fmt.Fprintln(os.Stderr, "                    刷机汇总")
	fmt.Fprintln(os.Stderr, strings.Repeat("=", 50))

	success, failed, skipped := 0, 0, 0
	for _, r := range results {
		if r.Success {
			success++
		} else if r.Error == "设备离线" {
			skipped++
		} else {
			failed++
		}
	}

	fmt.Fprintf(os.Stderr, "成功: %d 台\n", success)
	fmt.Fprintf(os.Stderr, "失败: %d 台\n", failed)
	fmt.Fprintf(os.Stderr, "跳过(离线): %d 台\n", skipped)
	fmt.Fprintf(os.Stderr, "总耗时: %.1f 分钟\n", totalTime.Minutes())

	if failed > 0 {
		fmt.Fprintln(os.Stderr, "\n失败设备:")
		for _, r := range results {
			if !r.Success && r.Error != "设备离线" {
				fmt.Fprintf(os.Stderr, "  - %s 通道%d (%s): %s\n", r.COM, r.Channel, r.IP, r.Error)
			}
		}
	}

	if skipped > 0 {
		fmt.Fprintln(os.Stderr, "\n跳过设备(离线):")
		for _, r := range results {
			if r.Error == "设备离线" {
				fmt.Fprintf(os.Stderr, "  - %s 通道%d (%s)\n", r.COM, r.Channel, r.IP)
			}
		}
	}

	fmt.Fprintln(os.Stderr, strings.Repeat("=", 50))
}

func confirm(prompt string) bool {
	fmt.Fprintf(os.Stderr, "%s (y/n): ", prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(strings.ToLower(input)) == "y"
}

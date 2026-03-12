package cmd

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// FRPC 下载地址
var frpcDownloadURLs = map[string]string{
	"windows/amd64": "https://jpy-1308564197.cos.ap-guangzhou.myqcloud.com/frpc.exe",
	"linux/amd64":   "https://jpy-1308564197.cos.ap-guangzhou.myqcloud.com/frp_0.61.1_linux_amd64",
	"linux/arm64":   "https://jpy-1308564197.cos.ap-guangzhou.myqcloud.com/frp_0.61.1_linux_arm64",
	"darwin/amd64":  "https://jpy-1308564197.cos.ap-guangzhou.myqcloud.com/frp_0.61.1_darwin_amd64",
	"darwin/arm64":  "https://jpy-1308564197.cos.ap-guangzhou.myqcloud.com/frp_0.61.1_darwin_arm64",
}

// 获取 FRPC 相关路径
func getFrpcPaths() (dir, binPath, configPath string) {
	home, _ := os.UserHomeDir()
	dir = filepath.Join(home, ".jpy", "frpc")
	if runtime.GOOS == "windows" {
		binPath = filepath.Join(dir, "frpc.exe")
	} else {
		binPath = filepath.Join(dir, "frpc")
	}
	configPath = filepath.Join(dir, "frpc.ini")
	return
}

// 自检入口
func runSelfCheck() {
	fmt.Println("========================================")
	fmt.Println("       JPY CLI 自检模式")
	fmt.Println("========================================")
	fmt.Println()

	// Windows 管理员权限检测
	if runtime.GOOS == "windows" {
		if !isWindowsAdmin() {
			fmt.Println("[!] 警告: 未以管理员身份运行")
			fmt.Println("    部分功能可能受限（如添加 Defender 白名单）")
			fmt.Println("    建议: 右键点击程序 -> 以管理员身份运行")
			fmt.Println()
		} else {
			fmt.Println("[✓] 已以管理员身份运行")
		}
	}

	// 检测 FRPC
	frpcDir, frpcBin, frpcConfig := getFrpcPaths()
	fmt.Printf("[1] FRPC 程序路径: %s\n", frpcBin)

	frpcExists := fileExists(frpcBin)
	configExists := fileExists(frpcConfig)

	if frpcExists {
		fmt.Println("    状态: 已安装")
	} else {
		fmt.Println("    状态: 未安装")
	}

	fmt.Printf("[2] FRPC 配置文件: %s\n", frpcConfig)
	if configExists {
		fmt.Println("    状态: 已配置")
		// 显示当前配置摘要
		showConfigSummary(frpcConfig)
	} else {
		fmt.Println("    状态: 未配置")
	}

	// 检测 FRPC 运行状态
	frpcRunning := isFrpcRunning()
	fmt.Printf("[3] FRPC 运行状态: ")
	if frpcRunning {
		fmt.Println("运行中")
	} else {
		fmt.Println("未运行")
	}

	// 检测本地 9090 端口
	fmt.Printf("[4] 本地 9090 端口: ")
	if checkPort("127.0.0.1", 9090) {
		fmt.Println("可用 (jpy server 运行中)")
	} else {
		fmt.Println("不可用 (jpy server 未运行)")
	}

	fmt.Println()
	fmt.Println("========================================")

	// 根据状态提供选项
	if !frpcExists || !configExists {
		fmt.Println()
		fmt.Println("检测到 FRPC 未完全配置，是否进入配置向导？")
		fmt.Print("输入 y 继续，其他键退出: ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "y" || input == "yes" {
			runFrpcSetup(frpcDir, frpcBin, frpcConfig, frpcExists)
		}
	} else if !frpcRunning {
		fmt.Println()
		fmt.Print("FRPC 未运行，是否启动？(y/n): ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "y" || input == "yes" {
			startFrpc(frpcBin, frpcConfig)
		}
	} else {
		fmt.Println()
		fmt.Println("一切正常！")
	}
}

// FRPC 配置向导
func runFrpcSetup(frpcDir, frpcBin, frpcConfig string, frpcExists bool) {
	fmt.Println()
	fmt.Println("========== FRPC 配置向导 ==========")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// 1. 下载 FRPC（如果不存在）
	if !frpcExists {
		fmt.Println("[步骤 1/5] 下载 FRPC")

		// Windows: 先添加 Defender 白名单
		if runtime.GOOS == "windows" && isWindowsAdmin() {
			fmt.Println("  正在添加 Defender 白名单...")
			addDefenderExclusion(frpcDir)
		}

		// 创建目录
		if err := os.MkdirAll(frpcDir, 0755); err != nil {
			fmt.Printf("  创建目录失败: %v\n", err)
			return
		}

		// 下载
		platform := runtime.GOOS + "/" + runtime.GOARCH
		url, ok := frpcDownloadURLs[platform]
		if !ok {
			fmt.Printf("  不支持的平台: %s\n", platform)
			return
		}

		fmt.Printf("  下载地址: %s\n", url)
		fmt.Printf("  保存路径: %s\n", frpcBin)

		if err := downloadFile(url, frpcBin); err != nil {
			fmt.Printf("  下载失败: %v\n", err)
			return
		}

		// 设置可执行权限
		if runtime.GOOS != "windows" {
			os.Chmod(frpcBin, 0755)
		}

		fmt.Println("  下载完成!")
	} else {
		fmt.Println("[步骤 1/5] FRPC 已存在，跳过下载")
	}

	fmt.Println()

	// 2. 输入服务器地址
	fmt.Println("[步骤 2/5] 输入 FRPS 服务器地址")
	fmt.Print("  服务器地址 (如 frp.example.com): ")
	serverAddr, _ := reader.ReadString('\n')
	serverAddr = strings.TrimSpace(serverAddr)
	if serverAddr == "" {
		fmt.Println("  服务器地址不能为空")
		return
	}

	// 3. 输入服务器端口
	fmt.Println()
	fmt.Println("[步骤 3/5] 输入 FRPS 服务器端口")
	fmt.Print("  服务器端口 (默认 7000): ")
	serverPort, _ := reader.ReadString('\n')
	serverPort = strings.TrimSpace(serverPort)
	if serverPort == "" {
		serverPort = "7000"
	}

	// 4. 输入密钥
	fmt.Println()
	fmt.Println("[步骤 4/5] 输入认证密钥")
	fmt.Print("  密钥 (token): ")
	token, _ := reader.ReadString('\n')
	token = strings.TrimSpace(token)
	if token == "" {
		fmt.Println("  密钥不能为空")
		return
	}

	// 5. 输入远程端口
	fmt.Println()
	fmt.Println("[步骤 5/5] 输入远程映射端口")
	fmt.Print("  远程端口 (如 20031): ")
	remotePort, _ := reader.ReadString('\n')
	remotePort = strings.TrimSpace(remotePort)
	if remotePort == "" {
		fmt.Println("  远程端口不能为空")
		return
	}

	// 生成配置文件（使用 ini 格式，兼容老版本 frpc）
	// 配置项名称使用 jpy-cli-{远程端口}，避免多个服务器使用同一 FRPS 时冲突
	config := fmt.Sprintf(`[common]
server_addr = %s
server_port = %s
token = %s

[jpy-cli-%s]
type = tcp
local_ip = 127.0.0.1
local_port = 9090
remote_port = %s
`, serverAddr, serverPort, token, remotePort, remotePort)

	if err := os.WriteFile(frpcConfig, []byte(config), 0644); err != nil {
		fmt.Printf("写入配置文件失败: %v\n", err)
		return
	}

	fmt.Println()
	fmt.Println("========== 配置完成 ==========")
	fmt.Println()
	fmt.Printf("配置文件: %s\n", frpcConfig)
	fmt.Println("配置内容:")
	fmt.Println("---")
	fmt.Println(config)
	fmt.Println("---")

	// 询问是否启动
	fmt.Print("是否立即启动 FRPC？(y/n): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "y" || input == "yes" {
		startFrpc(frpcBin, frpcConfig)
	}
}

// 启动 FRPC
func startFrpc(frpcBin, frpcConfig string) {
	fmt.Println()
	fmt.Println("正在启动 FRPC...")

	cmd := exec.Command(frpcBin, "-c", frpcConfig)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		fmt.Printf("启动失败: %v\n", err)
		return
	}

	fmt.Printf("FRPC 已启动 (PID: %d)\n", cmd.Process.Pid)
	fmt.Println("提示: FRPC 正在前台运行，按 Ctrl+C 停止")
	fmt.Println()

	// 等待进程结束
	cmd.Wait()
}

// 显示配置摘要
func showConfigSummary(configPath string) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "serverAddr") {
			fmt.Printf("    - %s\n", line)
		} else if strings.HasPrefix(line, "remotePort") {
			fmt.Printf("    - %s\n", line)
		}
	}
}

// 检测文件是否存在
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// 检测 FRPC 是否运行中
func isFrpcRunning() bool {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("tasklist", "/FI", "IMAGENAME eq frpc.exe", "/NH")
	} else {
		cmd = exec.Command("pgrep", "-x", "frpc")
	}

	output, err := cmd.Output()
	if err != nil {
		return false
	}

	if runtime.GOOS == "windows" {
		return strings.Contains(string(output), "frpc.exe")
	}
	return len(strings.TrimSpace(string(output))) > 0
}

// 检测端口是否可用
func checkPort(host string, port int) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s:%d/health", host, port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// 下载文件
func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	// 显示下载进度
	total := resp.ContentLength
	var downloaded int64

	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
			downloaded += int64(n)
			if total > 0 {
				fmt.Printf("\r  进度: %.1f%%", float64(downloaded)/float64(total)*100)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	fmt.Println()

	return nil
}

// Windows: 检测是否管理员权限
func isWindowsAdmin() bool {
	if runtime.GOOS != "windows" {
		return false
	}

	// 尝试打开一个需要管理员权限的文件
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil
}

// Windows: 添加 Defender 白名单
func addDefenderExclusion(path string) {
	if runtime.GOOS != "windows" {
		return
	}

	cmd := exec.Command("powershell", "-Command",
		fmt.Sprintf("Add-MpPreference -ExclusionPath '%s'", path))
	if err := cmd.Run(); err != nil {
		fmt.Printf("  添加白名单失败: %v\n", err)
	} else {
		fmt.Printf("  已添加白名单: %s\n", path)
	}
}

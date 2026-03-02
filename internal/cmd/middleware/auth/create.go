package auth

import (
	"bufio"
	"fmt"
	"jpy-cli/pkg/config"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func NewCreateCmd() *cobra.Command {
	var ipStr string
	var portStr string
	var username string
	var password string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "批量生成中间件服务器配置并添加到当前分组",
		RunE: func(cmd *cobra.Command, args []string) error {
			if ipStr != "" {
				return runCreateBatch(ipStr, portStr, username, password)
			}
			return runCreateInteractive()
		},
	}

	cmd.Flags().StringVarP(&ipStr, "ip", "i", "", "IP 范围，支持逗号分隔多个区间 (例如: 192.168.1.201-210,192.168.2.100)")
	cmd.Flags().StringVarP(&portStr, "port", "P", "443", "端口")
	cmd.Flags().StringVarP(&username, "username", "u", "admin", "用户名")
	cmd.Flags().StringVarP(&password, "password", "p", "admin", "密码")

	return cmd
}

func runCreateBatch(ipInput, portStr, username, password string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("加载配置失败: %v", err)
	}

	currentGroup := cfg.ActiveGroup
	if currentGroup == "" {
		currentGroup = "default"
	}

	// Parse comma-separated IP ranges
	rawRanges := strings.Split(ipInput, ",")
	var ipRanges []string
	for _, r := range rawRanges {
		r = strings.TrimSpace(r)
		if r != "" {
			ipRanges = append(ipRanges, r)
		}
	}

	if len(ipRanges) == 0 {
		return fmt.Errorf("未提供有效的 IP 范围")
	}

	return processCreate(cfg, currentGroup, ipRanges, portStr, username, password)
}

func runCreateInteractive() error {
	// Load config first to get current group
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("加载配置失败: %v", err)
	}

	// Default to "default" if ActiveGroup is empty
	currentGroup := cfg.ActiveGroup
	if currentGroup == "" {
		currentGroup = "default"
	}

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Printf("=== 批量添加服务器到当前分组 (%s) ===\n", currentGroup)
	fmt.Println("请输入 IP 区间 (每行一个，空行结束输入):")
	fmt.Println("格式示例: 192.168.23.201-210 或 192.168.23.201")

	var ipRanges []string
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}
		ipRanges = append(ipRanges, line)
	}

	if len(ipRanges) == 0 {
		return fmt.Errorf("未输入任何 IP 区间")
	}

	fmt.Print("请输入端口 (默认 443): ")
	scanner.Scan()
	portStr := strings.TrimSpace(scanner.Text())
	if portStr == "" {
		portStr = "443"
	}

	fmt.Print("请输入 Username (默认 admin): ")
	scanner.Scan()
	username := strings.TrimSpace(scanner.Text())
	if username == "" {
		username = "admin"
	}

	fmt.Print("请输入 Password (默认 admin): ")
	scanner.Scan()
	password := strings.TrimSpace(scanner.Text())
	if password == "" {
		password = "admin"
	}

	return processCreate(cfg, currentGroup, ipRanges, portStr, username, password)
}

func processCreate(cfg *config.Config, currentGroup string, ipRanges []string, portStr, username, password string) error {
	var newServers []config.LocalServerConfig
	var duplicateCount int
	var successCount int

	// Pre-load existing servers in current group for duplicate checking
	existingServers := make(map[string]bool)
	if groupServers, ok := cfg.Groups[currentGroup]; ok {
		for _, s := range groupServers {
			existingServers[s.URL] = true
		}
	}

	for _, ipRange := range ipRanges {
		ips, err := parseIPRange(ipRange)
		if err != nil {
			fmt.Printf("警告: 解析 IP 区间 '%s' 失败: %v\n", ipRange, err)
			continue
		}

		for _, ip := range ips {
			url := fmt.Sprintf("https://%s:%s", ip, portStr)

			// Check for duplicates
			if existingServers[url] {
				duplicateCount++
				continue
			}

			server := config.LocalServerConfig{
				URL:      url,
				Username: username,
				Password: password,
				Group:    currentGroup,
			}
			newServers = append(newServers, server)
			existingServers[url] = true // Mark as existing to avoid internal duplicates
			successCount++
		}
	}

	if len(newServers) > 0 {
		// Add to config
		for _, s := range newServers {
			config.AddServer(cfg, s)
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("保存配置失败: %v", err)
		}
		fmt.Printf("\n成功添加 %d 台服务器到分组 '%s'。\n", successCount, currentGroup)
	} else {
		fmt.Println("\n未添加任何服务器 (可能全部重复)。")
	}

	if duplicateCount > 0 {
		fmt.Printf("跳过重复服务器: %d\n", duplicateCount)
	}

	return nil
}

// Reuse parseIPRange logic (duplicated here to avoid cycle imports if we can't extract it easily yet)
func parseIPRange(input string) ([]string, error) {
	// Format: 192.168.23.201-210
	parts := strings.Split(input, ".")
	if len(parts) != 4 {
		return nil, fmt.Errorf("无效的 IP 格式")
	}

	lastPart := parts[3]
	if !strings.Contains(lastPart, "-") {
		// Single IP
		return []string{input}, nil
	}

	rangeParts := strings.Split(lastPart, "-")
	if len(rangeParts) != 2 {
		return nil, fmt.Errorf("无效的区间格式")
	}

	start, err := strconv.Atoi(rangeParts[0])
	if err != nil {
		return nil, fmt.Errorf("无效的起始数字")
	}
	end, err := strconv.Atoi(rangeParts[1])
	if err != nil {
		return nil, fmt.Errorf("无效的结束数字")
	}

	if start > end {
		return nil, fmt.Errorf("起始数字不能大于结束数字")
	}

	var ips []string
	baseIP := strings.Join(parts[:3], ".")
	for i := start; i <= end; i++ {
		ips = append(ips, fmt.Sprintf("%s.%d", baseIP, i))
	}

	return ips, nil
}

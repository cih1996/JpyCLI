package middleware

import (
	"bufio"
	"encoding/json"
	"fmt"
	"jpy-cli/pkg/config"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func NewCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "批量生成中间件服务器配置",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreateInteractive()
		},
	}
	return cmd
}

func runCreateInteractive() error {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("=== 批量生成中间件配置 ===")
	fmt.Println("请输入 IP 区间 (每行一个，空行结束输入):")
	fmt.Println("格式示例: 192.168.23.201-210")

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

	fmt.Print("请输入 Group (默认 local): ")
	scanner.Scan()
	group := strings.TrimSpace(scanner.Text())
	if group == "" {
		group = "local"
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

	var servers []config.LocalServerConfig

	for _, ipRange := range ipRanges {
		ips, err := parseIPRange(ipRange)
		if err != nil {
			fmt.Printf("警告: 解析 IP 区间 '%s' 失败: %v\n", ipRange, err)
			continue
		}

		for _, ip := range ips {
			url := fmt.Sprintf("https://%s:%s", ip, portStr)
			server := config.LocalServerConfig{
				URL:      url,
				Username: username,
				Password: password,
				Group:    group,
			}
			servers = append(servers, server)
		}
	}

	fmt.Printf("\n已生成 %d 个服务器配置。\n", len(servers))
	
	// Print JSON
	jsonData, err := json.MarshalIndent(servers, "", "  ")
	if err != nil {
		return fmt.Errorf("生成 JSON 失败: %v", err)
	}
	
	fmt.Println(string(jsonData))

	// Option to save to file
	fmt.Print("\n是否保存到文件? (y/n, 默认 n): ")
	scanner.Scan()
	save := strings.TrimSpace(scanner.Text())
	if strings.ToLower(save) == "y" {
		fmt.Print("请输入文件名 (默认 servers.json): ")
		scanner.Scan()
		filename := strings.TrimSpace(scanner.Text())
		if filename == "" {
			filename = "servers.json"
		}
		
		// Check if file exists to confirm overwrite/append?
		// User requirement is simple batch production. Overwrite is safest default for now unless we implement smart append.
		// Since servers.json is a JSON array, appending is tricky without parsing existing.
		
		err := os.WriteFile(filename, jsonData, 0644)
		if err != nil {
			return fmt.Errorf("写入文件失败: %v", err)
		}
		fmt.Printf("配置已保存到 %s\n", filename)
	}

	return nil
}

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

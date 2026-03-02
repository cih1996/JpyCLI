package middleware

import (
	"encoding/base64"
	"fmt"
	"jpy-cli/pkg/admin-middleware/api"
	"jpy-cli/pkg/admin-middleware/service"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

func NewSSHCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh [ip]",
		Short: "通过 SSH 连接中间件服务器 (自动获取 Root 密码)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ip := args[0]

			// 1. Validate IP (Must be direct IP, no port)
			if strings.Contains(ip, ":") {
				return fmt.Errorf("不支持带端口的地址: %s (仅支持直连IP)", ip)
			}
			parsedIP := net.ParseIP(ip)
			if parsedIP == nil {
				return fmt.Errorf("无效的 IP 地址: %s", ip)
			}

			// 2. Ensure Operation Login (for decrypting password)
			adminCfg, err := service.EnsureOperationLoggedIn()
			if err != nil {
				return err
			}

			// 3. Connect to get Banner (Key)
			fmt.Printf("正在连接 %s:22 获取密钥...\n", ip)
			bannerKey, err := getBannerKey(ip)
			if err != nil {
				return fmt.Errorf("获取密钥失败: %v", err)
			}
			fmt.Printf("获取到密钥: %s\n", bannerKey)

			// 4. Decrypt Password
			fmt.Println("正在解密 Root 密码...")
			client := api.NewClient(adminCfg.Token)
			rawPassword, err := client.DecryptPassword(bannerKey)
			if err != nil {
				return fmt.Errorf("解密密码失败: %v", err)
			}

			// Base64 Decode
			decodedBytes, err := base64.StdEncoding.DecodeString(rawPassword)
			if err != nil {
				return fmt.Errorf("Base64 解码失败: %v", err)
			}
			password := string(decodedBytes)

			fmt.Printf("密码解密成功: %s\n", password)

			// 5. Generate Connection Command
			printSSHCommand(ip, password)
			return nil
		},
	}
	return cmd
}

func printSSHCommand(ip, password string) {
	// Check if sshpass is installed
	if _, err := exec.LookPath("sshpass"); err == nil {
		fmt.Printf("\n检测到已安装 sshpass，可使用以下命令直接连接：\n")
		// Escape single quotes in password if necessary, though simpler to just warn.
		// Assuming password doesn't contain single quotes for now, or use double quotes.
		fmt.Printf("sshpass -p '%s' ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null root@%s\n", password, ip)
	} else {
		fmt.Printf("\n您可以复制以下命令进行连接：\n")
		fmt.Printf("ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null root@%s\n", ip)
		fmt.Printf("\n(连接时请粘贴上方解密后的密码)\n")
	}
}

func getBannerKey(ip string) (string, error) {
	var banner string

	config := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			// We don't have password yet, so this will likely fail Auth.
			// But we only need the banner.
			ssh.Password("dummy"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
		BannerCallback: func(message string) error {
			banner = strings.TrimSpace(message)
			return nil
		},
	}

	// We expect this to fail with "ssh: handshake failed: ssh: unable to authenticate"
	// But before that, BannerCallback should have been called.
	conn, err := ssh.Dial("tcp", ip+":22", config)
	if conn != nil {
		conn.Close()
	}

	// Check if we got the banner
	if banner != "" {
		// Clean up the banner.
		// The user example: "2Ruqp...root@192.168.0.102's password: "
		// The banner is likely just "2Ruqp..."
		// But ssh.Dial might fail *after* banner.

		// The banner might contain "root@<ip>'s password: " suffix if captured from raw stream,
		// but SSH BannerCallback usually returns the server's version string or pre-auth banner.
		// However, based on user input, the key is quite long.
		// Let's assume the banner IS the key.
		return banner, nil
	}

	// If we didn't get banner, maybe the error explains why
	if err != nil {
		// If error is auth failure, but banner is empty, then server didn't send banner?
		// Or maybe banner callback wasn't called.

		// It's possible the error message itself contains the banner if it's not a standard SSH banner
		// but some custom behavior. However, standard SSH libraries handle banners via callback.

		// Let's try to parse error if it's a specific type? No, usually not.
		return "", fmt.Errorf("连接失败或未收到密钥: %v", err)
	}

	return "", fmt.Errorf("未收到密钥")
}

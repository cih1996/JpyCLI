package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

func NewProxyCmd() *cobra.Command {
	var webPort int

	cmd := &cobra.Command{
		Use:   "proxy [url]",
		Short: "创建一个基于会话的透明代理",
		Long: `向运行中的 Web Server 申请一个代理 Token。
当访问返回的临时 URL 时，Web Server 会自动将当前会话切换为反向代理模式，
透明地转发所有流量到目标 URL，直到会话结束或手动退出。

注意：此命令需要在 Web Terminal 中运行，或者确保本地有运行在指定端口的 Web Server。`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetURL := args[0]

			// Try to detect port from env if not specified flag
			if webPort == 0 {
				// We can try to guess or use default 8080
				// Ideally, the Web Server should inject an env var when spawning the shell
				// But for now let's default to 8080
				webPort = 8080
			}

			// Request payload
			payload := map[string]string{
				"target": targetURL,
			}
			jsonData, _ := json.Marshal(payload)

			// Send request to local web server
			apiURL := fmt.Sprintf("http://127.0.0.1:%d/sys/proxy/create", webPort)
			resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
			if err != nil {
				return fmt.Errorf("无法连接到本地 Web Server (%s): %v\n请确保 Web Server 正在运行", apiURL, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(resp.Body)
				return fmt.Errorf("代理创建失败: %s", string(body))
			}

			var result map[string]string
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return fmt.Errorf("无法解析响应: %v", err)
			}

			token := result["token"]

			// Try to get current host from env or something?
			// Since we are in CLI, we might not know the external IP/Domain easily.
			// But we can just print the relative path or localhost.
			// The user is likely in the web terminal, so relative path is best.

			fmt.Printf("\n代理已就绪！\n")
			fmt.Printf("请点击或访问以下链接激活代理会话：\n\n")
			fmt.Printf("    /go/%s\n\n", token)
			fmt.Printf("激活后，Web Server 将进入混合代理模式：\n")
			fmt.Printf("- 访问 HTTP  -> 依然是 Web Terminal 或 HTTP 反向代理\n")
			fmt.Printf("- 访问 HTTPS -> 透明转发到 %s (解决 WSS/HTTP2 兼容性)\n", targetURL)
			fmt.Printf("\n如需退出代理模式，请务必访问 HTTP 地址： /jpy-exit\n")

			return nil
		},
	}

	cmd.Flags().IntVarP(&webPort, "port", "p", 0, "本地 Web Server 端口 (默认 8080)")

	return cmd
}

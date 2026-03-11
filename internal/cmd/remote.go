package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// extractRemoteFlag 从原始参数中提取 --remote 值和剩余参数
// 支持 --remote host:port 和 --remote=host:port
func extractRemoteFlag(args []string) (string, []string) {
	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--remote" {
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "--remote 需要指定地址 (host:port)\n")
				os.Exit(1)
			}
			addr := args[i+1]
			remaining := make([]string, 0, len(args)-2)
			remaining = append(remaining, args[:i]...)
			remaining = append(remaining, args[i+2:]...)
			return addr, remaining
		}

		if strings.HasPrefix(arg, "--remote=") {
			addr := strings.TrimPrefix(arg, "--remote=")
			if addr == "" {
				fmt.Fprintf(os.Stderr, "--remote 需要指定地址 (host:port)\n")
				os.Exit(1)
			}
			remaining := make([]string, 0, len(args)-1)
			remaining = append(remaining, args[:i]...)
			remaining = append(remaining, args[i+1:]...)
			return addr, remaining
		}
	}
	return "", nil
}

// remoteExec 将命令参数转发到远端 server 执行
func remoteExec(remoteAddr string, args []string) {
	if !strings.HasPrefix(remoteAddr, "http") {
		remoteAddr = "http://" + remoteAddr
	}
	url := strings.TrimRight(remoteAddr, "/") + "/exec"

	reqBody := execRequest{Args: args}
	body, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Fprintf(os.Stderr, "序列化请求失败: %v\n", err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "连接远端失败: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取响应失败: %v\n", err)
		os.Exit(1)
	}

	var result execResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		fmt.Fprintf(os.Stderr, "解析响应失败: %v\n%s\n", err, string(respBody))
		os.Exit(1)
	}

	if result.Stdout != "" {
		fmt.Fprint(os.Stdout, result.Stdout)
	}
	if result.Stderr != "" {
		fmt.Fprint(os.Stderr, result.Stderr)
	}

	os.Exit(result.ExitCode)
}

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
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

// extractAsyncFlag 从参数中提取 --async 和 --timeout 值
// 返回: isAsync, timeout(秒), 剩余参数
func extractAsyncFlag(args []string) (bool, int, []string) {
	isAsync := false
	timeout := 600 // 默认 10 分钟
	remaining := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--async" {
			isAsync = true
			continue
		}

		// --async-timeout N
		if arg == "--async-timeout" {
			if i+1 < len(args) {
				if t, err := strconv.Atoi(args[i+1]); err == nil {
					timeout = t
				}
				i++ // 跳过下一个参数
			}
			continue
		}

		// --async-timeout=N
		if strings.HasPrefix(arg, "--async-timeout=") {
			if t, err := strconv.Atoi(strings.TrimPrefix(arg, "--async-timeout=")); err == nil {
				timeout = t
			}
			continue
		}

		remaining = append(remaining, arg)
	}

	return isAsync, timeout, remaining
}

// asyncExecRequest 异步执行请求
type asyncExecRequest struct {
	Args    []string `json:"args"`
	Timeout int      `json:"timeout"`
}

// asyncExecResponse 异步执行响应
type asyncExecResponse struct {
	TaskID string `json:"task_id"`
	Status string `json:"status"`
}

// remoteExec 将命令参数转发到远端 server 执行
func remoteExec(remoteAddr string, args []string) {
	if !strings.HasPrefix(remoteAddr, "http") {
		remoteAddr = "http://" + remoteAddr
	}

	// 检查是否异步模式
	isAsync, timeout, cleanArgs := extractAsyncFlag(args)

	if isAsync {
		remoteExecAsync(remoteAddr, cleanArgs, timeout)
		return
	}

	url := strings.TrimRight(remoteAddr, "/") + "/exec"

	reqBody := execRequest{Args: cleanArgs}
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

// remoteExecAsync 异步执行命令
func remoteExecAsync(remoteAddr string, args []string, timeout int) {
	url := strings.TrimRight(remoteAddr, "/") + "/exec/async"

	reqBody := asyncExecRequest{Args: args, Timeout: timeout}
	body, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Fprintf(os.Stderr, "序列化请求失败: %v\n", err)
		os.Exit(1)
	}

	client := &http.Client{Timeout: 30 * time.Second}
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

	var result asyncExecResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		fmt.Fprintf(os.Stderr, "解析响应失败: %v\n%s\n", err, string(respBody))
		os.Exit(1)
	}

	// 输出任务信息
	fmt.Printf("任务已提交\n")
	fmt.Printf("Task ID: %s\n", result.TaskID)
	fmt.Printf("Status: %s\n", result.Status)
	fmt.Printf("Timeout: %d 秒\n", timeout)
	fmt.Printf("\n查看进度:\n")
	fmt.Printf("  jpy shell --remote %s --task %s\n", strings.TrimPrefix(strings.TrimPrefix(remoteAddr, "http://"), "https://"), result.TaskID)
	fmt.Printf("\n终止任务:\n")
	fmt.Printf("  jpy shell --remote %s --kill %s\n", strings.TrimPrefix(strings.TrimPrefix(remoteAddr, "http://"), "https://"), result.TaskID)
}

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

// extractRemoteFlags 从参数中提取 --async、--async-timeout、--timeout 值
// 返回: isAsync, asyncTimeout(秒), syncTimeout(秒), 剩余参数
func extractRemoteFlags(args []string) (bool, int, int, []string) {
	isAsync := false
	asyncTimeout := 600 // 异步默认 10 分钟
	syncTimeout := 120  // 同步默认 2 分钟
	remaining := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--async" {
			isAsync = true
			continue
		}

		// --async-timeout N (异步任务超时)
		if arg == "--async-timeout" {
			if i+1 < len(args) {
				if t, err := strconv.Atoi(args[i+1]); err == nil {
					asyncTimeout = t
				}
				i++ // 跳过下一个参数
			}
			continue
		}

		// --async-timeout=N
		if strings.HasPrefix(arg, "--async-timeout=") {
			if t, err := strconv.Atoi(strings.TrimPrefix(arg, "--async-timeout=")); err == nil {
				asyncTimeout = t
			}
			continue
		}

		// --timeout N (同步 HTTP 超时)
		if arg == "--timeout" {
			if i+1 < len(args) {
				if t, err := strconv.Atoi(args[i+1]); err == nil {
					syncTimeout = t
				}
				i++ // 跳过下一个参数
			}
			continue
		}

		// --timeout=N
		if strings.HasPrefix(arg, "--timeout=") {
			if t, err := strconv.Atoi(strings.TrimPrefix(arg, "--timeout=")); err == nil {
				syncTimeout = t
			}
			continue
		}

		remaining = append(remaining, arg)
	}

	return isAsync, asyncTimeout, syncTimeout, remaining
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

	// 提取所有远程执行相关参数
	isAsync, asyncTimeout, syncTimeout, cleanArgs := extractRemoteFlags(args)

	if isAsync {
		remoteExecAsync(remoteAddr, cleanArgs, asyncTimeout)
		return
	}

	url := strings.TrimRight(remoteAddr, "/") + "/exec"

	reqBody := execRequest{Args: cleanArgs}
	body, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Fprintf(os.Stderr, "序列化请求失败: %v\n", err)
		os.Exit(1)
	}

	// syncTimeout=0 表示无限等待
	var client *http.Client
	if syncTimeout == 0 {
		client = &http.Client{} // 无超时
	} else {
		client = &http.Client{Timeout: time.Duration(syncTimeout) * time.Second}
	}
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
	if timeout == 0 {
		fmt.Printf("Timeout: 无限\n")
	} else {
		fmt.Printf("Timeout: %d 秒\n", timeout)
	}
	fmt.Printf("\n查看进度:\n")
	fmt.Printf("  jpy shell --remote %s --task %s\n", strings.TrimPrefix(strings.TrimPrefix(remoteAddr, "http://"), "https://"), result.TaskID)
	fmt.Printf("\n终止任务:\n")
	fmt.Printf("  jpy shell --remote %s --kill %s\n", strings.TrimPrefix(strings.TrimPrefix(remoteAddr, "http://"), "https://"), result.TaskID)
}

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newShellCmd() *cobra.Command {
	var (
		cmdStr  string
		timeout int
		async   bool
		taskID  string
		kill    string
		tasks   bool
		output  string
	)

	cmd := &cobra.Command{
		Use:   "shell",
		Short: "远程执行系统 shell 命令（需配合 --remote 使用）",
		Long: `在远端机器上执行系统 shell 命令。
同步模式：等待命令完成，返回输出。
异步模式：提交后台任务，返回 task_id，后续查询结果。

本命令必须配合 --remote 使用，本地执行无意义。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// 从 root 拿 --remote 地址（shell 命令强制要求 --remote）
			// 但实际上 --remote 在 Cobra 之前就被拦截了，所以这里需要特殊处理
			// shell 命令本地也可以跑（直接执行），但主要场景是远程
			if tasks {
				return fmt.Errorf("--tasks 仅在 --remote 模式下可用")
			}
			if taskID != "" {
				return fmt.Errorf("--task 仅在 --remote 模式下可用")
			}
			if kill != "" {
				return fmt.Errorf("--kill 仅在 --remote 模式下可用")
			}

			// 本地执行
			if cmdStr == "" && len(args) > 0 {
				cmdStr = strings.Join(args, " ")
			}
			if cmdStr == "" {
				return fmt.Errorf("必须指定 -c <命令>")
			}

			timeoutDur := 120 * time.Second
			if timeout > 0 {
				timeoutDur = time.Duration(timeout) * time.Second
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), timeoutDur)
			defer cancel()

			shellC := shellCmd(ctx, cmdStr)
			shellC.Stdout = os.Stdout
			shellC.Stderr = os.Stderr

			if err := shellC.Run(); err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					os.Exit(exitErr.ExitCode())
				}
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&cmdStr, "cmd", "c", "", "要执行的 shell 命令")
	cmd.Flags().IntVar(&timeout, "timeout", 120, "超时秒数（默认 120，异步默认 600）")
	cmd.Flags().BoolVar(&async, "async", false, "异步模式：提交后台任务")
	cmd.Flags().StringVar(&taskID, "task", "", "查询指定任务的状态和日志")
	cmd.Flags().StringVar(&kill, "kill", "", "终止指定任务")
	cmd.Flags().BoolVar(&tasks, "tasks", false, "列出所有任务")
	cmd.Flags().StringVarP(&output, "output", "o", "plain", "输出模式: plain/json")

	return cmd
}

// remoteShellExec 处理 shell 命令的远程执行（在 --remote 拦截时调用）
func remoteShellExec(remoteAddr string, args []string) {
	if !strings.HasPrefix(remoteAddr, "http") {
		remoteAddr = "http://" + remoteAddr
	}
	baseURL := strings.TrimRight(remoteAddr, "/")

	// 解析 shell 子命令的参数
	var (
		cmdStr  string
		timeout int
		async   bool
		taskID  string
		kill    string
		tasks   bool
		output  string = "plain"
	)

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-c", "--cmd":
			if i+1 < len(args) {
				cmdStr = args[i+1]
				i++
			}
		case "--timeout":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &timeout)
				i++
			}
		case "--async":
			async = true
		case "--task":
			if i+1 < len(args) {
				taskID = args[i+1]
				i++
			}
		case "--kill":
			if i+1 < len(args) {
				kill = args[i+1]
				i++
			}
		case "--tasks":
			tasks = true
		case "-o", "--output":
			if i+1 < len(args) {
				output = args[i+1]
				i++
			}
		default:
			// 位置参数作为命令
			if !strings.HasPrefix(args[i], "-") && cmdStr == "" {
				cmdStr = args[i]
			}
		}
	}

	// 列出所有任务
	if tasks {
		resp := httpGet(baseURL + "/shell/tasks")
		if output == "json" {
			fmt.Println(string(resp))
		} else {
			var taskList []taskResponse
			json.Unmarshal(resp, &taskList)
			if len(taskList) == 0 {
				fmt.Println("无任务")
			} else {
				fmt.Println("TASK_ID\tSTATUS\tEXIT\tELAPSED\tCOMMAND")
				for _, t := range taskList {
					fmt.Printf("%s\t%s\t%d\t%s\t%s\n",
						t.TaskID, t.Status, t.ExitCode, t.Elapsed, t.Command)
				}
			}
		}
		return
	}

	// 查询任务
	if taskID != "" {
		resp := httpGet(baseURL + "/shell/task?id=" + taskID)
		if output == "json" {
			fmt.Println(string(resp))
		} else {
			var t taskResponse
			json.Unmarshal(resp, &t)
			if t.TaskID == "" {
				// 可能是错误
				fmt.Fprintln(os.Stderr, string(resp))
				os.Exit(1)
			}
			fmt.Printf("任务: %s  状态: %s  退出码: %d  耗时: %s\n",
				t.TaskID, t.Status, t.ExitCode, t.Elapsed)
			fmt.Printf("命令: %s\n", t.Command)
			if t.Stdout != "" {
				fmt.Print(t.Stdout)
			}
			if t.Stderr != "" {
				fmt.Fprint(os.Stderr, t.Stderr)
			}
		}
		return
	}

	// 终止任务
	if kill != "" {
		resp := httpGet(baseURL + "/shell/kill?id=" + kill)
		if output == "json" {
			fmt.Println(string(resp))
		} else {
			var t taskResponse
			json.Unmarshal(resp, &t)
			fmt.Printf("任务 %s 已终止\n", t.TaskID)
		}
		return
	}

	// 执行命令
	if cmdStr == "" {
		fmt.Fprintln(os.Stderr, "必须指定 -c <命令>")
		os.Exit(1)
	}

	if async {
		// 异步提交
		asyncTimeout := 600
		if timeout > 0 {
			asyncTimeout = timeout
		}
		body, _ := json.Marshal(shellRequest{Cmd: cmdStr, Timeout: asyncTimeout})
		resp := httpPost(baseURL+"/shell/async", body)
		if output == "json" {
			fmt.Println(string(resp))
		} else {
			var ar asyncResponse
			json.Unmarshal(resp, &ar)
			fmt.Printf("任务已提交: %s (状态: %s)\n", ar.TaskID, ar.Status)
			fmt.Printf("查询: jpy shell --remote %s --task %s\n",
				strings.TrimPrefix(baseURL, "http://"), ar.TaskID)
		}
		return
	}

	// 同步执行
	syncTimeout := 120
	if timeout > 0 {
		syncTimeout = timeout
	}
	body, _ := json.Marshal(shellRequest{Cmd: cmdStr, Timeout: syncTimeout})

	client := &http.Client{Timeout: time.Duration(syncTimeout+10) * time.Second}
	resp, err := client.Post(baseURL+"/shell", "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "连接远端失败: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if output == "json" {
		fmt.Println(string(respBody))
	} else {
		var result execResponse
		json.Unmarshal(respBody, &result)
		if result.Stdout != "" {
			fmt.Print(result.Stdout)
		}
		if result.Stderr != "" {
			fmt.Fprint(os.Stderr, result.Stderr)
		}
		os.Exit(result.ExitCode)
	}
}

func httpGet(url string) []byte {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "请求失败: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return body
}

func httpPost(url string, body []byte) []byte {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(os.Stderr, "请求失败: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return data
}

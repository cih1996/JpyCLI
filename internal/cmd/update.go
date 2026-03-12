package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var remote string

	cmd := &cobra.Command{
		Use:   "update <本地文件路径|远程URL>",
		Short: "更新 CLI 程序",
		Long: `更新 CLI 程序到新版本。

支持两种方式：
1. 本地文件：jpy update ./jpy-new.exe --remote 192.168.1.100:9090
   将本地文件上传到远程并更新
2. 远程URL：jpy update https://example.com/jpy.exe --remote 192.168.1.100:9090
   让远程从 URL 下载并更新

更新流程：
1. 下载/上传新版本到临时目录
2. 验证新版本可执行
3. 替换当前程序（Windows 使用延迟替换）
4. 重启服务（如果在 server 模式运行）`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			source := args[0]

			if remote == "" {
				// 本地更新
				return updateLocal(source)
			}

			// 远程更新
			return updateRemote(remote, source)
		},
	}

	cmd.Flags().StringVar(&remote, "remote", "", "远程服务器地址 (host:port)")

	return cmd
}

// 本地更新
func updateLocal(source string) error {
	fmt.Println("本地更新暂不支持，请使用 --remote 参数进行远程更新")
	return nil
}

// 远程更新
func updateRemote(remote, source string) error {
	if !strings.Contains(remote, "://") {
		remote = "http://" + remote
	}

	// 判断是本地文件还是远程 URL
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return updateRemoteFromURL(remote, source)
	}

	return updateRemoteFromFile(remote, source)
}

// 从本地文件更新远程
func updateRemoteFromFile(remote, localPath string) error {
	// 检查本地文件是否存在
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		return fmt.Errorf("本地文件不存在: %s", localPath)
	}

	fmt.Printf("正在上传文件到远程服务器...\n")
	fmt.Printf("  本地文件: %s\n", localPath)
	fmt.Printf("  远程地址: %s\n", remote)

	// 1. 上传文件到远程临时目录
	tempPath, err := uploadFileToRemote(remote, localPath)
	if err != nil {
		return fmt.Errorf("上传文件失败: %v", err)
	}
	fmt.Printf("  上传完成: %s\n", tempPath)

	// 2. 执行更新
	return executeRemoteUpdate(remote, tempPath)
}

// 从 URL 更新远程
func updateRemoteFromURL(remote, url string) error {
	fmt.Printf("正在让远程服务器下载文件...\n")
	fmt.Printf("  下载地址: %s\n", url)
	fmt.Printf("  远程地址: %s\n", remote)

	// 1. 让远程下载文件
	tempPath, err := downloadFileOnRemote(remote, url)
	if err != nil {
		return fmt.Errorf("远程下载失败: %v", err)
	}
	fmt.Printf("  下载完成: %s\n", tempPath)

	// 2. 执行更新
	return executeRemoteUpdate(remote, tempPath)
}

// 上传文件到远程
func uploadFileToRemote(remote, localPath string) (string, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// 构建 multipart 请求
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// 添加文件
	part, err := writer.CreateFormFile("file", filepath.Base(localPath))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", err
	}

	// 添加目标路径（临时目录）
	tempName := fmt.Sprintf("jpy-update-%d%s", time.Now().Unix(), getExeSuffix())
	writer.WriteField("dest", tempName)
	writer.Close()

	// 发送请求
	resp, err := http.Post(remote+"/file/upload", writer.FormDataContentType(), &buf)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Path    string `json:"path"`
		Error   string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if !result.Success {
		return "", fmt.Errorf("%s", result.Error)
	}

	return result.Path, nil
}

// 让远程下载文件
func downloadFileOnRemote(remote, url string) (string, error) {
	tempName := fmt.Sprintf("jpy-update-%d%s", time.Now().Unix(), getExeSuffix())

	reqBody, _ := json.Marshal(map[string]string{
		"url":  url,
		"dest": tempName,
	})

	resp, err := http.Post(remote+"/file/download", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Success bool   `json:"success"`
		Path    string `json:"path"`
		Error   string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if !result.Success {
		return "", fmt.Errorf("%s", result.Error)
	}

	return result.Path, nil
}

// 执行远程更新
func executeRemoteUpdate(remote, newFilePath string) error {
	fmt.Println("正在执行更新...")

	// 获取远程系统信息
	resp, err := http.Get(remote + "/version")
	if err != nil {
		return fmt.Errorf("获取远程版本信息失败: %v", err)
	}
	defer resp.Body.Close()

	var versionInfo struct {
		Success bool `json:"success"`
		Data    struct {
			Platform string `json:"platform"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&versionInfo); err != nil {
		return fmt.Errorf("解析版本信息失败: %v", err)
	}

	isWindows := strings.HasPrefix(versionInfo.Data.Platform, "windows")

	// 获取当前可执行文件路径
	// 通过 /shell 执行命令获取
	var currentPath string
	if isWindows {
		currentPath, err = getRemoteExePath(remote, "echo %~dp0%~nx0")
		if err != nil {
			// 备用方案：使用固定路径
			currentPath = "C:\\jpy\\jpy.exe"
		}
	} else {
		currentPath, err = getRemoteExePath(remote, "readlink -f /proc/$$/exe 2>/dev/null || echo /usr/local/bin/jpy")
		if err != nil {
			currentPath = "/usr/local/bin/jpy"
		}
	}

	fmt.Printf("  当前程序路径: %s\n", currentPath)
	fmt.Printf("  新版本路径: %s\n", newFilePath)

	// 构建更新命令
	var updateCmd string
	if isWindows {
		// Windows: 使用 cmd /c 延迟替换
		// 1. 等待 2 秒让当前进程退出
		// 2. 复制新文件覆盖旧文件
		// 3. 删除临时文件
		// 4. 重启服务
		updateCmd = fmt.Sprintf(
			`cmd /c "timeout /t 2 /nobreak >nul && copy /y "%s" "%s" && del "%s" && "%s" server"`,
			newFilePath, currentPath, newFilePath, currentPath,
		)
	} else {
		// Linux/macOS: 直接替换
		updateCmd = fmt.Sprintf(
			`sleep 2 && cp "%s" "%s" && chmod +x "%s" && rm "%s" && "%s" server &`,
			newFilePath, currentPath, currentPath, newFilePath, currentPath,
		)
	}

	// 异步执行更新命令
	reqBody, _ := json.Marshal(map[string]interface{}{
		"cmd":     updateCmd,
		"timeout": 60,
	})

	resp2, err := http.Post(remote+"/shell/async", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("执行更新命令失败: %v", err)
	}
	defer resp2.Body.Close()

	var asyncResult struct {
		TaskID string `json:"task_id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&asyncResult); err != nil {
		return fmt.Errorf("解析更新结果失败: %v", err)
	}

	fmt.Println()
	fmt.Println("========== 更新已启动 ==========")
	fmt.Printf("任务 ID: %s\n", asyncResult.TaskID)
	fmt.Println("远程服务将在 2 秒后重启...")
	fmt.Println("请稍后检查远程服务状态：")
	fmt.Printf("  curl %s/health\n", remote)
	fmt.Printf("  curl %s/version\n", remote)

	return nil
}

// 获取远程可执行文件路径
func getRemoteExePath(remote, cmd string) (string, error) {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"cmd":     cmd,
		"timeout": 5,
	})

	resp, err := http.Post(remote+"/shell", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		ExitCode int    `json:"exit_code"`
		Stdout   string `json:"stdout"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return strings.TrimSpace(result.Stdout), nil
}

// 获取可执行文件后缀
func getExeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

package file

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type pushResponse struct {
	Success bool   `json:"success"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Error   string `json:"error,omitempty"`
}

func newPushCmd() *cobra.Command {
	var (
		remoteAddr string
		destPath   string
		output     string
		timeout    int
	)

	cmd := &cobra.Command{
		Use:   "push <local-file>",
		Short: "上传本地文件到远程机器",
		Long: `上传本地文件到远程 jpy server。

支持大文件传输（最大 5GB），使用流式上传避免内存溢出。

示例:
  # 上传文件到远程默认目录
  jpy file push ./rom.zip --remote 192.168.1.100:9090

  # 上传到指定路径
  jpy file push ./rom.zip --remote 192.168.1.100:9090 --dest D:\flash\rom.zip

  # 大文件设置更长超时（秒）
  jpy file push ./large.zip --remote 192.168.1.100:9090 --timeout 3600`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			localFile := args[0]

			if remoteAddr == "" {
				return fmt.Errorf("必须指定 --remote 参数")
			}

			// 打开文件（流式读取）
			f, err := os.Open(localFile)
			if err != nil {
				return fmt.Errorf("打开文件失败: %v", err)
			}
			defer f.Close()

			stat, err := f.Stat()
			if err != nil {
				return fmt.Errorf("获取文件信息失败: %v", err)
			}

			filename := filepath.Base(localFile)
			if destPath == "" {
				destPath = filename
			}

			// 使用 pipe 实现流式上传，避免大文件占用内存
			pr, pw := io.Pipe()
			writer := multipart.NewWriter(pw)

			// 异步写入 multipart 数据
			errCh := make(chan error, 1)
			go func() {
				defer pw.Close()
				defer writer.Close()

				part, err := writer.CreateFormFile("file", filename)
				if err != nil {
					errCh <- err
					return
				}

				// 流式复制文件内容
				if _, err := io.Copy(part, f); err != nil {
					errCh <- err
					return
				}

				// 添加目标路径
				if err := writer.WriteField("dest", destPath); err != nil {
					errCh <- err
					return
				}

				errCh <- nil
			}()

			// 发送请求
			url := normalizeURL(remoteAddr) + "/file/upload"
			req, err := http.NewRequest("POST", url, pr)
			if err != nil {
				return fmt.Errorf("创建请求失败: %v", err)
			}
			req.Header.Set("Content-Type", writer.FormDataContentType())

			// 显示进度
			fmt.Fprintf(os.Stderr, "上传中: %s (%.2f MB)...\n", filename, float64(stat.Size())/(1024*1024))

			client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
			resp, err := client.Do(req)

			// 先等待写入完成，获取可能的写入错误
			writeErr := <-errCh

			if err != nil {
				// 如果是写入错误导致的请求失败，优先报告写入错误
				if writeErr != nil {
					return fmt.Errorf("上传失败: %v", writeErr)
				}
				return fmt.Errorf("上传失败: %v", err)
			}
			defer resp.Body.Close()

			// 检查写入错误
			if writeErr != nil {
				return fmt.Errorf("写入数据失败: %v", writeErr)
			}

			respBody, _ := io.ReadAll(resp.Body)
			var result pushResponse
			if err := json.Unmarshal(respBody, &result); err != nil {
				return fmt.Errorf("解析响应失败: %s", string(respBody))
			}

			if output == "json" {
				fmt.Println(string(respBody))
			} else {
				if result.Success {
					fmt.Printf("OK\t%s\t%d bytes\n", result.Path, result.Size)
				} else {
					fmt.Fprintf(os.Stderr, "FAIL\t%s\n", result.Error)
					os.Exit(1)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&remoteAddr, "remote", "", "远程 jpy server 地址（必填）")
	cmd.Flags().StringVar(&destPath, "dest", "", "远程目标路径（默认使用原文件名）")
	cmd.Flags().StringVarP(&output, "output", "o", "plain", "输出模式: plain/json")
	cmd.Flags().IntVar(&timeout, "timeout", 1800, "上传超时时间（秒），默认30分钟")

	return cmd
}

func normalizeURL(addr string) string {
	if !strings.HasPrefix(addr, "http") {
		addr = "http://" + addr
	}
	return strings.TrimRight(addr, "/")
}

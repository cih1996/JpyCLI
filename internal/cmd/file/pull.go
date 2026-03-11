package file

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

type pullRequest struct {
	URL  string `json:"url"`
	Dest string `json:"dest"`
}

type pullResponse struct {
	Success bool   `json:"success"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	Error   string `json:"error,omitempty"`
}

func newPullCmd() *cobra.Command {
	var (
		remoteAddr string
		destPath   string
		output     string
		timeout    int
	)

	cmd := &cobra.Command{
		Use:   "pull <url>",
		Short: "让远程机器从 URL 下载文件",
		Long: `让远程 jpy server 从指定 URL 下载文件到本地。

示例:
  # 远程下载文件到默认目录
  jpy file pull "https://example.com/rom.zip" --remote 192.168.1.100:9090

  # 下载到指定路径
  jpy file pull "https://example.com/rom.zip" --remote 192.168.1.100:9090 --dest D:\flash\rom.zip

  # 设置超时时间（秒）
  jpy file pull "https://example.com/rom.zip" --remote 192.168.1.100:9090 --timeout 600`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			downloadURL := args[0]

			if remoteAddr == "" {
				return fmt.Errorf("必须指定 --remote 参数")
			}

			reqBody := pullRequest{
				URL:  downloadURL,
				Dest: destPath,
			}

			body, _ := json.Marshal(reqBody)
			url := normalizeURL(remoteAddr) + "/file/download"

			client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
			resp, err := client.Post(url, "application/json", bytes.NewReader(body))
			if err != nil {
				return fmt.Errorf("请求失败: %v", err)
			}
			defer resp.Body.Close()

			respBody, _ := io.ReadAll(resp.Body)
			var result pullResponse
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
	cmd.Flags().StringVar(&destPath, "dest", "", "远程保存路径（默认使用 URL 文件名）")
	cmd.Flags().StringVarP(&output, "output", "o", "plain", "输出模式: plain/json")
	cmd.Flags().IntVar(&timeout, "timeout", 600, "下载超时时间（秒）")

	return cmd
}

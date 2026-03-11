package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// 版本信息（编译时注入）
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

type versionInfo struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	GitCommit string `json:"git_commit"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

type remoteVersionResp struct {
	Success bool        `json:"success"`
	Data    versionInfo `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func newVersionCmd() *cobra.Command {
	var (
		remoteAddr string
		output     string
	)

	cmd := &cobra.Command{
		Use:   "version",
		Short: "显示版本信息",
		Long: `显示 jpy-cli 版本信息。

示例:
  # 查看本地版本
  jpy version

  # 查看远程 server 版本
  jpy version --remote 192.168.1.100:9090

  # JSON 格式输出
  jpy version -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if remoteAddr != "" {
				return showRemoteVersion(remoteAddr, output)
			}
			return showLocalVersion(output)
		},
	}

	cmd.Flags().StringVar(&remoteAddr, "remote", "", "查询远程 jpy server 版本")
	cmd.Flags().StringVarP(&output, "output", "o", "plain", "输出模式: plain/json")

	return cmd
}

func showLocalVersion(output string) error {
	info := versionInfo{
		Version:   Version,
		BuildTime: BuildTime,
		GitCommit: GitCommit,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}

	if output == "json" {
		data, _ := json.Marshal(info)
		fmt.Println(string(data))
	} else {
		fmt.Printf("jpy-cli %s\n", info.Version)
		fmt.Printf("Build:    %s\n", info.BuildTime)
		fmt.Printf("Commit:   %s\n", info.GitCommit)
		fmt.Printf("Go:       %s\n", info.GoVersion)
		fmt.Printf("Platform: %s\n", info.Platform)
	}
	return nil
}

func showRemoteVersion(addr, output string) error {
	url := normalizeRemoteURL(addr) + "/version"

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("连接远程失败: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result remoteVersionResp
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("解析响应失败: %s", string(body))
	}

	if !result.Success {
		return fmt.Errorf("远程错误: %s", result.Error)
	}

	if output == "json" {
		data, _ := json.Marshal(result.Data)
		fmt.Println(string(data))
	} else {
		fmt.Printf("Remote: %s\n", addr)
		fmt.Printf("jpy-cli %s\n", result.Data.Version)
		fmt.Printf("Build:    %s\n", result.Data.BuildTime)
		fmt.Printf("Commit:   %s\n", result.Data.GitCommit)
		fmt.Printf("Go:       %s\n", result.Data.GoVersion)
		fmt.Printf("Platform: %s\n", result.Data.Platform)
	}
	return nil
}

func normalizeRemoteURL(addr string) string {
	if !strings.HasPrefix(addr, "http") {
		addr = "http://" + addr
	}
	return strings.TrimRight(addr, "/")
}

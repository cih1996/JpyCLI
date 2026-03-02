package middleware

import (
	"encoding/json"
	"fmt"
	"jpy-cli/pkg/config"
	"jpy-cli/pkg/middleware/connector"
	"os"
	"sync"

	"github.com/spf13/cobra"
)

func NewReloginCmd() *cobra.Command {
	var outputMode string

	cmd := &cobra.Command{
		Use:   "relogin",
		Short: "尝试重新连接已软删除的服务器",
		Long: `遍历当前分组中被"暂时移除" (Disabled) 的服务器，尝试重新连接。
如果连接成功，则自动恢复（取消软删除）。

输出模式:
  --output tui     默认输出
  --output plain   纯文本，适合脚本
  --output json    JSON 格式，适合程序解析`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			activeGroup := cfg.ActiveGroup
			if activeGroup == "" {
				activeGroup = "default"
			}
			servers := cfg.Groups[activeGroup]

			var disabledIndices []int
			for i, s := range servers {
				if s.Disabled {
					disabledIndices = append(disabledIndices, i)
				}
			}

			if len(disabledIndices) == 0 {
				if outputMode == "json" {
					fmt.Println(`{"total":0,"restored":0,"failed":0,"message":"当前分组没有被暂时移除的服务器"}`)
					return nil
				}
				fmt.Println("当前分组没有被暂时移除的服务器。")
				return nil
			}

			if outputMode == "plain" || outputMode == "tui" {
				msgTarget := os.Stdout
				if outputMode == "plain" {
					msgTarget = os.Stderr
				}
				fmt.Fprintf(msgTarget, "发现 %d 台被移除的服务器，开始重新连接检测...\n", len(disabledIndices))
			}

			var wg sync.WaitGroup
			var mu sync.Mutex
			sem := make(chan struct{}, 20)

			type reloginResult struct {
				URL     string `json:"url"`
				Status  string `json:"status"`
				Error   string `json:"error,omitempty"`
			}

			var results []reloginResult
			successCount := 0
			failCount := 0

			for _, idx := range disabledIndices {
				wg.Add(1)
				go func(i int) {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()

					s := &cfg.Groups[activeGroup][i]
					conn := connector.NewConnectorService(cfg)
					ws, err := conn.Connect(*s)

					mu.Lock()
					defer mu.Unlock()

					if err != nil {
						failCount++
						results = append(results, reloginResult{
							URL:    cleanServerURL(s.URL),
							Status: "failed",
							Error:  err.Error(),
						})
						if outputMode == "tui" {
							fmt.Printf("[失败] %s: %v\n", s.URL, err)
						}
					} else {
						ws.Close()
						successCount++
						s.Disabled = false
						results = append(results, reloginResult{
							URL:    cleanServerURL(s.URL),
							Status: "restored",
						})
						if outputMode == "tui" {
							fmt.Printf("[恢复] %s 连接成功，已恢复。\n", s.URL)
						}
					}
				}(idx)
			}
			wg.Wait()

			if successCount > 0 {
				if err := config.Save(cfg); err != nil {
					return err
				}
			}

			switch outputMode {
			case "json":
				out := map[string]interface{}{
					"total":    len(disabledIndices),
					"restored": successCount,
					"failed":   failCount,
					"results":  results,
				}
				data, _ := json.MarshalIndent(out, "", "  ")
				fmt.Println(string(data))
			case "plain":
				fmt.Println("URL\tSTATUS\tERROR")
				for _, r := range results {
					fmt.Printf("%s\t%s\t%s\n", r.URL, r.Status, r.Error)
				}
				fmt.Fprintf(os.Stderr, "总计: %d, 恢复: %d, 失败: %d\n", len(disabledIndices), successCount, failCount)
			default:
				fmt.Printf("\n完成。成功恢复: %d, 依旧失败: %d\n", successCount, failCount)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputMode, "output", "o", "tui", "输出模式 (tui/plain/json)")
	return cmd
}

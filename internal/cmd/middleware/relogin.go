package middleware

import (
	"fmt"
	"jpy-cli/pkg/config"
	"jpy-cli/pkg/middleware/connector"
	"sync"

	"github.com/spf13/cobra"
)

func NewReloginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relogin",
		Short: "尝试重新连接已软删除的服务器",
		Long:  `遍历当前分组中被“暂时移除” (Disabled) 的服务器，尝试重新连接。如果连接成功，则自动恢复（取消软删除）。`,
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
				fmt.Println("当前分组没有被暂时移除的服务器。")
				return nil
			}

			fmt.Printf("发现 %d 台被移除的服务器，开始重新连接检测...\n", len(disabledIndices))

			var wg sync.WaitGroup
			var mu sync.Mutex
			sem := make(chan struct{}, 20)
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
					if err != nil {
						mu.Lock()
						failCount++
						mu.Unlock()
						// Log failure
						fmt.Printf("[失败] %s: %v\n", s.URL, err)
					} else {
						ws.Close()
						mu.Lock()
						successCount++
						s.Disabled = false // Restore
						mu.Unlock()
						fmt.Printf("[恢复] %s 连接成功，已恢复。\n", s.URL)
					}
				}(idx)
			}
			wg.Wait()

			fmt.Printf("\n完成。成功恢复: %d, 依旧失败: %d\n", successCount, failCount)

			if successCount > 0 {
				return config.Save(cfg)
			}
			return nil
		},
	}
	return cmd
}

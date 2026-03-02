package middleware

import (
	"bufio"
	"fmt"
	"jpy-cli/pkg/config"
	"jpy-cli/pkg/middleware/connector"
	"jpy-cli/pkg/middleware/device/selector"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

func NewRemoveCmd() *cobra.Command {
	var removeAll bool
	var hasError bool
	var force bool
	var search string

	cmd := &cobra.Command{
		Use:   "remove",
		Short: "移除/软删除中间件服务器",
		Long: `从当前分组中移除中间件服务器。
支持软删除（默认，可恢复）和强制删除（--force）。
可以根据错误状态、名称匹配或批量操作。`,
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
			if len(servers) == 0 {
				fmt.Println("当前分组没有服务器。")
				return nil
			}

			var targets []*config.LocalServerConfig
			var indices []int

			// 1. Identify Targets
			if removeAll {
				for i := range servers {
					targets = append(targets, &servers[i])
					indices = append(indices, i)
				}
			} else if hasError {
				fmt.Println("正在检测连接状态 (只删除连接失败的)...")
				failedIndices := checkConnectionFailures(servers, cfg)
				for _, idx := range failedIndices {
					targets = append(targets, &servers[idx])
					indices = append(indices, idx)
				}
			} else if search != "" {
				for i, s := range servers {
					if selector.MatchServerPattern(s.URL, search) || selector.MatchServerPattern(s.Username, search) {
						targets = append(targets, &servers[i])
						indices = append(indices, i)
					}
				}
			} else {
				// Interactive mode
				fmt.Println("进入交互模式...")
				fmt.Printf("当前分组 (%s) 服务器列表:\n", activeGroup)
				for i, s := range servers {
					status := "正常"
					if s.Disabled {
						status = "已移除"
					} else if s.LastLoginError != "" {
						status = "错误: " + s.LastLoginError
					}
					fmt.Printf("[%d] %s (%s)\n", i+1, s.URL, status)
				}
				fmt.Println("\n请输入要移除的序号(逗号分隔)，或输入指令:")
				fmt.Println(" a: 所有 (all)")
				fmt.Println(" e: 仅错误 (error)")
				fmt.Println(" q: 退出")

				reader := bufio.NewReader(os.Stdin)
				fmt.Print("> ")
				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(input)

				if input == "q" || input == "" {
					return nil
				} else if input == "a" {
					for i := range servers {
						targets = append(targets, &servers[i])
						indices = append(indices, i)
					}
				} else if input == "e" {
					fmt.Println("正在检测连接状态...")
					failedIndices := checkConnectionFailures(servers, cfg)
					for _, idx := range failedIndices {
						targets = append(targets, &servers[idx])
						indices = append(indices, idx)
					}
				} else {
					parts := strings.Split(input, ",")
					for _, p := range parts {
						var idx int
						if _, err := fmt.Sscanf(p, "%d", &idx); err == nil {
							if idx > 0 && idx <= len(servers) {
								targets = append(targets, &servers[idx-1])
								indices = append(indices, idx-1)
							}
						}
					}
				}
			}

			if len(targets) == 0 {
				fmt.Println("未找到匹配的服务器。")
				return nil
			}

			// 2. Confirm (--all + --force 组合时跳过交互确认，适配自动化场景)
			fmt.Printf("即将%s %d 台服务器:\n", func() string {
				if force {
					return "永久删除"
				}
				return "暂时移除(软删除)"
			}(), len(targets))

			for i, t := range targets {
				if i >= 10 {
					fmt.Printf("... 等共 %d 台\n", len(targets))
					break
				}
				fmt.Printf(" - %s\n", t.URL)
			}

			skipConfirm := (removeAll || hasError || search != "") && force
			if !skipConfirm && !confirmAction() {
				return nil
			}

			// 3. Execute
			if force {
				// Remove from slice (backwards to avoid index shift issues if we were deleting by index,
				// but here we need to rebuild the slice or map carefully)
				// Easiest way: filter keep list
				var keep []config.LocalServerConfig
				targetMap := make(map[string]bool)
				for _, t := range targets {
					targetMap[t.URL] = true
				}

				for _, s := range servers {
					if !targetMap[s.URL] {
						keep = append(keep, s)
					}
				}
				cfg.Groups[activeGroup] = keep
				fmt.Printf("已永久删除 %d 台服务器。\n", len(targets))
			} else {
				// Soft delete: Update Disabled flag
				// We need to update the actual items in the slice.
				// Pointers in 'targets' point to elements in 'servers' slice?
				// CAUTION: 'servers' is a copy of slice header? No, 'servers := cfg.Groups[activeGroup]' copies slice header.
				// Elements share underlying array.
				// But appending to 'servers' might reallocate. Here we just modify.
				// But we range over 'servers' (value copy) earlier?
				// No: 'for i := range servers { targets = append(targets, &servers[i]) }'
				// This works if 'servers' is not reallocated.
				// Let's rely on URL matching to be safe.
				count := 0
				for i := range cfg.Groups[activeGroup] {
					s := &cfg.Groups[activeGroup][i]
					for _, t := range targets {
						if s.URL == t.URL {
							s.Disabled = true
							count++
							break
						}
					}
				}
				fmt.Printf("已暂时移除 %d 台服务器。\n", count)
			}

			return config.Save(cfg)
		},
	}

	cmd.Flags().BoolVar(&removeAll, "all", false, "删除当前分组内的所有服务器")
	cmd.Flags().BoolVar(&hasError, "has-error", false, "只删除连接失败的服务器")
	cmd.Flags().BoolVar(&force, "force", false, "永久删除 (不提供则为软删除)")
	cmd.Flags().StringVar(&search, "search", "", "模糊匹配删除")

	return cmd
}

func checkConnectionFailures(servers []config.LocalServerConfig, cfg *config.Config) []int {
	var failedIndices []int
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 20)

	for i, s := range servers {
		if s.Disabled {
			continue // Already disabled
		}
		wg.Add(1)
		go func(idx int, server config.LocalServerConfig) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			conn := connector.NewConnectorService(cfg)
			// Short timeout for check
			// We might need to enforce a shorter timeout here specifically?
			// The global timeout applies.
			ws, err := conn.Connect(server)
			if err != nil {
				mu.Lock()
				failedIndices = append(failedIndices, idx)
				mu.Unlock()
				fmt.Printf("检测失败: %s (%v)\n", server.URL, err)
			} else {
				ws.Close()
			}
		}(i, s)
	}
	wg.Wait()
	return failedIndices
}

func confirmAction() bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("确认执行? [y/N]: ")
	text, _ := reader.ReadString('\n')
	return strings.ToLower(strings.TrimSpace(text)) == "y"
}

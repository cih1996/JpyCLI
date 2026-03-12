package com

import (
	"encoding/json"
	"fmt"
	"jpy-cli/pkg/comport"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func NewRestartCmd() *cobra.Command {
	var channels string

	cmd := &cobra.Command{
		Use:   "restart",
		Short: "重启通道",
		Long: `重启指定通道。

示例:
  # 重启单个通道
  jpy com restart --port COM3 --channel 1

  # 重启多个通道
  jpy com restart --port COM3 --channel 1,2,3

  # 重启通道范围
  jpy com restart --port COM3 --channel 2-20

  # 重启所有通道
  jpy com restart --port COM3 --channel 0
  jpy com restart --port COM3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := comport.ResolvePort(flagPort, "com restart")
			if err != nil {
				return err
			}

			// 解析通道列表
			chList := parseChannelList(channels)
			if len(chList) == 0 {
				return fmt.Errorf("无效的通道参数: %s", channels)
			}

			// 构建重启任务
			items := make([]comport.RestartItem, len(chList))
			for i, ch := range chList {
				items[i] = comport.RestartItem{Channel: ch}
			}

			results, err := comport.RunRestartBatch(port, items, false, flagSkipConnect)
			if err != nil {
				return err
			}

			success, failed := 0, 0
			for _, r := range results {
				if r.Success {
					success++
				} else {
					failed++
				}
			}

			switch flagOutput {
			case "json":
				b, _ := json.Marshal(map[string]interface{}{
					"port": port, "channels": chList,
					"total": len(results), "success": success, "failed": failed,
					"results": results,
				})
				fmt.Println(string(b))
			default:
				for _, r := range results {
					status := "OK"
					if !r.Success {
						status = "FAIL:" + r.Error
					}
					fmt.Printf("%s\t%d\t%s\n", port, r.Channel, status)
				}
				fmt.Fprintf(os.Stderr, "--- total: %d, success: %d, failed: %d\n", len(results), success, failed)
			}

			if failed > 0 {
				os.Exit(1)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&channels, "channel", "0", "通道: 单个(1)、列表(1,2,3)、范围(2-20)、所有(0)")

	return cmd
}

// parseChannelList 解析通道参数，支持: 0(所有), 1(单个), 1,2,3(列表), 2-20(范围)
func parseChannelList(ch string) []int {
	ch = strings.TrimSpace(ch)
	if ch == "" || ch == "0" {
		// 0 表示所有通道
		return []int{0}
	}

	result := make([]int, 0)

	// 支持逗号分隔
	for _, p := range strings.Split(ch, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		// 支持范围 (如 2-20)
		if strings.Contains(p, "-") {
			parts := strings.Split(p, "-")
			if len(parts) == 2 {
				start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
				end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err1 == nil && err2 == nil && start > 0 && end >= start {
					for i := start; i <= end; i++ {
						result = append(result, i)
					}
				}
			}
		} else {
			// 单个数字
			if n, err := strconv.Atoi(p); err == nil && n >= 0 {
				result = append(result, n)
			}
		}
	}

	return result
}

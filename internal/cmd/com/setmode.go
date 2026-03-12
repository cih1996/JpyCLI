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

func NewSetModeCmd() *cobra.Command {
	var (
		mode     string
		channels string
	)

	cmd := &cobra.Command{
		Use:   "set-mode",
		Short: "设置通道工作模式 (hub/otg)",
		Long: `设置通道工作模式。

示例:
  # 设置单个通道
  jpy com set-mode --port COM3 --mode hub --channel 1

  # 设置多个通道
  jpy com set-mode --port COM3 --mode otg --channel 1,2,3

  # 设置通道范围
  jpy com set-mode --port COM3 --mode hub --channel 2-20

  # 设置所有通道
  jpy com set-mode --port COM3 --mode otg --channel 0
  jpy com set-mode --port COM3 --mode otg`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if mode == "" {
				return fmt.Errorf("必须指定 --mode 参数 (hub/otg)")
			}
			m := strings.ToLower(mode)
			if m != "hub" && m != "otg" {
				return fmt.Errorf("无效模式: %s (请使用 hub 或 otg)", mode)
			}

			port, err := comport.ResolvePort(flagPort, "com set-mode")
			if err != nil {
				return err
			}

			// 解析通道列表
			chList := parseSetModeChannelList(channels)
			if len(chList) == 0 {
				return fmt.Errorf("无效的通道参数: %s", channels)
			}

			// 构建任务
			items := make([]comport.SetModeItem, len(chList))
			for i, ch := range chList {
				items[i] = comport.SetModeItem{Mode: m, Channel: ch}
			}

			results, err := comport.RunSetModeBatch(port, items, false, flagSkipConnect)
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
					"port": port, "channels": chList, "mode": m,
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

	cmd.Flags().StringVarP(&mode, "mode", "m", "", "工作模式: hub/otg（必填）")
	cmd.Flags().StringVar(&channels, "channel", "0", "通道: 单个(1)、列表(1,2,3)、范围(2-20)、所有(0)")

	return cmd
}

// parseSetModeChannelList 解析通道参数
func parseSetModeChannelList(ch string) []int {
	ch = strings.TrimSpace(ch)
	if ch == "" || ch == "0" {
		return []int{0}
	}

	result := make([]int, 0)
	for _, p := range strings.Split(ch, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
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
			if n, err := strconv.Atoi(p); err == nil && n >= 0 {
				result = append(result, n)
			}
		}
	}
	return result
}

package com

import (
	"encoding/json"
	"fmt"
	"jpy-cli/pkg/comport"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func NewSetModeCmd() *cobra.Command {
	var (
		mode    string
		channel int
	)

	cmd := &cobra.Command{
		Use:   "set-mode",
		Short: "设置通道工作模式 (hub/otg)",
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

			results, err := comport.RunSetModeBatch(port, []comport.SetModeItem{{Mode: m, Channel: channel}}, false, flagSkipConnect)
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
					"port": port, "channel": channel, "mode": m,
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
	cmd.Flags().IntVar(&channel, "channel", 0, "通道号 (1-20，0=所有通道)")

	return cmd
}

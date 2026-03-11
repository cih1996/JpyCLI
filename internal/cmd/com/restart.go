package com

import (
	"encoding/json"
	"fmt"
	"jpy-cli/pkg/comport"
	"os"

	"github.com/spf13/cobra"
)

func NewRestartCmd() *cobra.Command {
	var channel int

	cmd := &cobra.Command{
		Use:   "restart",
		Short: "重启通道",
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := comport.ResolvePort(flagPort, "com restart")
			if err != nil {
				return err
			}

			results, err := comport.RunRestartBatch(port, []comport.RestartItem{{Channel: channel}}, false, flagSkipConnect)
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
					"port": port, "channel": channel,
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

	cmd.Flags().IntVar(&channel, "channel", 0, "通道号 (1-20，0=所有通道)")

	return cmd
}

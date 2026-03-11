package device

import (
	"context"
	"encoding/json"
	"fmt"
	"jpy-cli/pkg/middleware/connector"
	"jpy-cli/pkg/middleware/device/status"
	"strings"

	"github.com/spf13/cobra"
)

func NewStatusCmd() *cobra.Command {
	var detail bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "查看服务器和设备状态概览",
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := resolveCredentials()
			if err != nil {
				return err
			}
			server := toServerInfo(creds)

			results, err := status.GetServerStatusStats(
				context.Background(),
				[]connector.ServerInfo{server},
				status.StatusFilters{Detail: detail},
				nil,
			)
			if err != nil {
				return err
			}

			switch flagOutput {
			case "json":
				b, _ := json.Marshal(results)
				fmt.Println(string(b))
			default:
				fmt.Println("SERVER\tSTATUS\tLICENSE\tDEVICES\tBIZ\tIP\tADB\tUSB\tOTG\tSPEED")
				for _, r := range results {
					cleanURL := strings.TrimPrefix(r.ServerURL, "https://")
					cleanURL = strings.TrimPrefix(cleanURL, "http://")
					speed := "-"
					if r.NetworkSpeed != "" {
						speed = r.NetworkSpeed
					}
					fmt.Printf("%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%s\n",
						cleanURL, r.Status, r.LicenseStatus,
						r.DeviceCount, r.BizOnlineCount, r.IPCount,
						r.ADBCount, r.USBCount, r.OTGCount, speed)
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&detail, "detail", false, "显示详细信息（固件版本、网速等）")

	return cmd
}

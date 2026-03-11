package device

import (
	"encoding/json"
	"fmt"
	"jpy-cli/pkg/middleware/connector"
	"jpy-cli/pkg/middleware/device/selector"
	"jpy-cli/pkg/middleware/model"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func NewListCmd() *cobra.Command {
	var (
		filterIP   string
		filterUUID string
		filterSeat int
		limit      int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出设备详细状态",
		RunE: func(cmd *cobra.Command, args []string) error {
			creds, err := resolveCredentials()
			if err != nil {
				return err
			}

			server := toServerInfo(creds)
			devices, err := selector.SelectDevices(selector.SelectionOptions{
				Servers: []connector.ServerInfo{server},
				IP:      filterIP,
				UUID:    filterUUID,
				Seat:    filterSeat,
			})
			if err != nil {
				return err
			}

			if len(devices) == 0 {
				if flagOutput == "json" {
					fmt.Println(`{"total":0,"devices":[]}`)
				} else {
					fmt.Println("没有找到设备。")
				}
				return nil
			}

			sort.Slice(devices, func(i, j int) bool {
				return devices[i].Seat < devices[j].Seat
			})

			if limit > 0 && len(devices) > limit {
				devices = devices[:limit]
			}

			switch flagOutput {
			case "json":
				printListJSON(devices)
			default:
				printListPlain(devices)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&filterIP, "ip", "", "按设备 IP 过滤（模糊匹配）")
	cmd.Flags().StringVar(&filterUUID, "uuid", "", "按设备 UUID 过滤（模糊匹配）")
	cmd.Flags().IntVar(&filterSeat, "seat", -1, "按机位号过滤")
	cmd.Flags().IntVarP(&limit, "limit", "l", 0, "限制显示数量")

	return cmd
}

func printListPlain(devices []model.DeviceInfo) {
	// Header
	fmt.Println("SERVER\tSEAT\tUUID\tMODEL\tANDROID\tONLINE\tBIZ\tIP\tADB\tUSB")
	for _, d := range devices {
		cleanURL := strings.TrimPrefix(d.ServerURL, "https://")
		cleanURL = strings.TrimPrefix(cleanURL, "http://")
		fmt.Printf("%s\t%d\t%s\t%s\t%s\t%v\t%v\t%s\t%v\t%v\n",
			cleanURL, d.Seat, d.UUID, d.Model, d.Android,
			d.IsOnline, d.BizOnline, d.IP, d.ADBEnabled, d.USBMode)
	}
	fmt.Printf("--- total: %d\n", len(devices))
}

func printListJSON(devices []model.DeviceInfo) {
	type jsonDevice struct {
		Server  string `json:"server"`
		Seat    int    `json:"seat"`
		UUID    string `json:"uuid"`
		Model   string `json:"model"`
		Android string `json:"android"`
		Online  bool   `json:"online"`
		Biz     bool   `json:"biz_online"`
		IP      string `json:"ip"`
		ADB     bool   `json:"adb"`
		USB     bool   `json:"usb"`
	}
	type jsonOut struct {
		Total   int          `json:"total"`
		Devices []jsonDevice `json:"devices"`
	}

	out := jsonOut{Total: len(devices)}
	for _, d := range devices {
		out.Devices = append(out.Devices, jsonDevice{
			Server:  d.ServerURL,
			Seat:    d.Seat,
			UUID:    d.UUID,
			Model:   d.Model,
			Android: d.Android,
			Online:  d.IsOnline,
			Biz:     d.BizOnline,
			IP:      d.IP,
			ADB:     d.ADBEnabled,
			USB:     d.USBMode,
		})
	}
	b, _ := json.Marshal(out)
	fmt.Println(string(b))
}

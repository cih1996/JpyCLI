package device

import (
	"fmt"
	"jpy-cli/pkg/middleware/device/selector"
	"jpy-cli/pkg/middleware/model"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

func NewListCmd() *cobra.Command {
	var limit int
	opts := CommonFlags{}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出设备详细状态",
		Long: `列出所有设备的详细状态信息。

输出模式:
  --output tui     带颜色的格式化表格（默认）
  --output plain   纯文本制表符分隔，适合 SSH / grep / awk / AI
  --output json    JSON 格式输出，适合程序对接`,
		RunE: func(cmd *cobra.Command, args []string) error {
			selOpts, err := opts.ToSelectorOptions()
			if err != nil {
				return err
			}

			selOpts.Interactive = false
			// 非 TUI 模式下跳过 BubbleTea 进度条，防止 Win7 SSH 崩溃
			if opts.Output == "plain" || opts.Output == "json" {
				selOpts.Silent = true
			}

			devices, err := selector.SelectDevices(selOpts)
			if err != nil {
				return err
			}

			if len(devices) == 0 {
				if opts.Output == "json" {
					fmt.Println("{\"total\":0,\"devices\":[]}")
					return nil
				}
				fmt.Println("没有找到符合条件的设备。")
				return nil
			}

			// Sort: Server Order -> Seat
			sort.Slice(devices, func(i, j int) bool {
				if devices[i].ServerIndex != devices[j].ServerIndex {
					return devices[i].ServerIndex < devices[j].ServerIndex
				}
				return devices[i].Seat < devices[j].Seat
			})

			displayDevices := devices
			if limit > 0 && len(displayDevices) > limit {
				displayDevices = displayDevices[:limit]
			}

			switch opts.Output {
			case "json":
				printDeviceListJSON(displayDevices)
			case "plain":
				printDeviceListPlain(displayDevices)
				if limit > 0 && len(devices) > limit {
					fmt.Printf("--- 仅显示前 %d 条，共 %d 台\n", limit, len(devices))
				}
			default:
				printListTUITable(devices, displayDevices, limit)
			}

			return nil
		},
	}

	AddCommonFlags(cmd, &opts)
	cmd.Flags().IntVarP(&limit, "limit", "l", 100, "限制显示数量 (默认 100)")

	return cmd
}

// printListTUITable 使用 lipgloss 渲染设备列表表格（原始 TUI 输出）
func printListTUITable(allDevices, displayDevices []model.DeviceInfo, limit int) {
	var (
		headerStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				Align(lipgloss.Center)

		cellStyle = lipgloss.NewStyle().
				Align(lipgloss.Center)

		onlineStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
		offlineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

		headers = []string{"服务器", "机位", "序列号", "型号", "安卓", "状态", "业务", "IP", "ADB", "模式"}
		widths  = []int{24, 6, 22, 10, 8, 8, 6, 16, 6, 6}
	)

	cleanServerURL := func(url string) string {
		url = strings.TrimPrefix(url, "https://")
		url = strings.TrimPrefix(url, "http://")
		return url
	}

	var headerRow string
	for i, h := range headers {
		headerRow = lipgloss.JoinHorizontal(lipgloss.Top, headerRow, headerStyle.Width(widths[i]).Render(h))
	}
	fmt.Println(headerRow)
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("-", lipgloss.Width(headerRow))))

	var statsTotal, statsOnline, statsBiz, statsADB, statsUSB, statsOTG int
	for _, d := range allDevices {
		statsTotal++
		if d.IsOnline {
			statsOnline++
		}
		if d.BizOnline {
			statsBiz++
		}
		if d.ADBEnabled {
			statsADB++
		}
		if d.USBMode {
			statsUSB++
		} else {
			statsOTG++
		}
	}

	for _, d := range displayDevices {
		stStatus := offlineStyle.Render("离线")
		if d.IsOnline {
			stStatus = onlineStyle.Render("在线")
		}

		stBiz := offlineStyle.Render("否")
		if d.BizOnline {
			stBiz = onlineStyle.Render("是")
		}

		stADB := offlineStyle.Render("关闭")
		if d.ADBEnabled {
			stADB = onlineStyle.Render("开启")
		}

		stMode := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render("OTG")
		if d.USBMode {
			stMode = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render("USB")
		}

		row := []string{
			cellStyle.Width(widths[0]).Render(cleanServerURL(d.ServerURL)),
			cellStyle.Width(widths[1]).Render(fmt.Sprintf("%d", d.Seat)),
			cellStyle.Width(widths[2]).Render(d.UUID),
			cellStyle.Width(widths[3]).Render(d.Model),
			cellStyle.Width(widths[4]).Render(d.Android),
			lipgloss.NewStyle().Width(widths[5]).Align(lipgloss.Center).Render(stStatus),
			lipgloss.NewStyle().Width(widths[6]).Align(lipgloss.Center).Render(stBiz),
			cellStyle.Width(widths[7]).Render(d.IP),
			lipgloss.NewStyle().Width(widths[8]).Align(lipgloss.Center).Render(stADB),
			lipgloss.NewStyle().Width(widths[9]).Align(lipgloss.Center).Render(stMode),
		}

		var rowStr string
		for _, cell := range row {
			rowStr = lipgloss.JoinHorizontal(lipgloss.Top, rowStr, cell)
		}
		fmt.Println(rowStr)
	}

	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("-", lipgloss.Width(headerRow))))

	summary := fmt.Sprintf("总计: %d 台 | 在线: %d | 业务在线: %d | ADB开启: %d | USB: %d | OTG: %d",
		statsTotal, statsOnline, statsBiz, statsADB, statsUSB, statsOTG)

	if limit > 0 && statsTotal > limit {
		summary += fmt.Sprintf(" (仅显示前 %d 条)", limit)
	}

	fmt.Println(lipgloss.NewStyle().Bold(true).Render(summary))
}

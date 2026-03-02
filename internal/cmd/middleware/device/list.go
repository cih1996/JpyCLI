package device

import (
	"fmt"
	"jpy-cli/pkg/middleware/device/selector"
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
		RunE: func(cmd *cobra.Command, args []string) error {
			selOpts, err := opts.ToSelectorOptions()
			if err != nil {
				return err
			}

			// Force interactive off for list command
			selOpts.Interactive = false

			devices, err := selector.SelectDevices(selOpts)
			if err != nil {
				return err
			}

			if len(devices) == 0 {
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

			// Apply limit for display only
			displayDevices := devices
			if limit > 0 && len(displayDevices) > limit {
				displayDevices = displayDevices[:limit]
			}

			// Output Table with Lipgloss
			var (
				headerStyle = lipgloss.NewStyle().
						Bold(true).
						Foreground(lipgloss.Color("205")).
						Align(lipgloss.Center)

				cellStyle = lipgloss.NewStyle().
						Align(lipgloss.Center)

				// Status Styles
				onlineStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))  // Green
				offlineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240")) // Grey

				headers = []string{"服务器", "机位", "序列号", "型号", "安卓", "状态", "业务", "IP", "ADB", "模式"}
				widths  = []int{24, 6, 22, 10, 8, 8, 6, 16, 6, 6}
			)

			// Helper to clean URL
			cleanURL := func(url string) string {
				url = strings.TrimPrefix(url, "https://")
				url = strings.TrimPrefix(url, "http://")
				return url
			}

			// Render Header
			var headerRow string
			for i, h := range headers {
				headerRow = lipgloss.JoinHorizontal(lipgloss.Top, headerRow, headerStyle.Width(widths[i]).Render(h))
			}
			fmt.Println(headerRow)
			fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("-", lipgloss.Width(headerRow))))

			// Statistics
			var (
				statsTotal  int
				statsOnline int
				statsBiz    int
				statsADB    int
				statsUSB    int
				statsOTG    int
			)

			for _, d := range devices {
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
				// Determine styles
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
					cellStyle.Width(widths[0]).Render(cleanURL(d.ServerURL)),
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

			// Summary Footer
			summary := fmt.Sprintf("总计: %d 台 | 在线: %d | 业务在线: %d | ADB开启: %d | USB: %d | OTG: %d",
				statsTotal, statsOnline, statsBiz, statsADB, statsUSB, statsOTG)

			// If limit applied, show note
			if limit > 0 && statsTotal > limit {
				summary += fmt.Sprintf(" (仅显示前 %d 条)", limit)
			}

			fmt.Println(lipgloss.NewStyle().Bold(true).Render(summary))

			return nil
		},
	}

	AddCommonFlags(cmd, &opts)
	cmd.Flags().IntVarP(&limit, "limit", "l", 100, "限制显示数量 (默认 100)")

	return cmd
}

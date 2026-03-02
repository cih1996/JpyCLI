package device

import (
	"fmt"
	httpclient "jpy-cli/pkg/client/http"
	wsclient "jpy-cli/pkg/client/ws"
	"jpy-cli/pkg/config"
	"jpy-cli/pkg/logger"
	"jpy-cli/pkg/middleware/device/api"
	"jpy-cli/pkg/middleware/device/selector"
	"jpy-cli/pkg/middleware/model"
	"jpy-cli/pkg/tui"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type ServerStatusStats struct {
	ServerURL       string
	Status          string
	LicenseStatus   string
	SN              string
	ControlAddr     string
	LicenseName     string
	FirmwareVersion string
	NetworkSpeed    string
	NetworkSpeedVal float64 // For filtering
	DeviceCount     int
	BizOnlineCount  int
	IPCount         int
	UUIDCount       int
	ADBCount        int
	USBCount        int
	OTGCount        int
	Error           error
	OrderIndex      int
}

func NewStatusCmd() *cobra.Command {
	opts := CommonFlags{}
	var detail bool
	var (
		bizOnlineGT        int
		bizOnlineLT        int
		ipCountGT          int
		ipCountLT          int
		uuidCountGT        int
		uuidCountLT        int
		snGT               string
		snLT               string
		authFailed         bool
		clusterContains    string
		clusterNotContains string
		fwVersionHas       string
		fwVersionNot       string
		netSpeedGT         float64
		netSpeedLT         float64
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "查看设备状态和服务器健康状况",
		Long: `查看中间件服务器的在线状态和聚合设备统计信息。

输出模式:
  --output tui     交互式进度条（默认）
  --output plain   纯文本表格，无 ANSI 转义码，适合 SSH / 脚本 / AI
  --output json    JSON 格式输出，适合程序对接

支持使用统一的筛选参数来快速定位问题服务器或特定状态的设备。
筛选条件可以组合使用（AND逻辑）。`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("无法加载配置: %v", err)
			}

			output := opts.Output

			// Determine group
			targetGroup := opts.Group
			if targetGroup == "" {
				targetGroup = cfg.ActiveGroup
			}
			if targetGroup == "" {
				targetGroup = "default"
			}

			if output == "tui" {
				fmt.Printf("当前分组: %s\n", targetGroup)
			}
			logger.Infof("开始检查分组设备状态: %s", targetGroup)

			// Get servers for the target group
			servers := config.GetGroupServers(cfg, targetGroup)

			// Map server URL to its index for sorting
			serverOrder := make(map[string]int)
			for i, s := range servers {
				serverOrder[s.URL] = i
			}

			// Filter servers by search term if provided
			var targets []config.LocalServerConfig
			for _, s := range servers {
				if s.Disabled {
					continue
				}
				if selector.MatchServerPattern(s.URL, opts.ServerPattern) {
					targets = append(targets, s)
				}
			}

			if len(targets) == 0 {
				if output == "json" {
					fmt.Println("{\"summary\":{\"total_servers\":0},\"servers\":[]}")
					return nil
				}
				return fmt.Errorf("分组 '%s' 中未找到匹配的服务器", targetGroup)
			}

			concurrency := config.GlobalSettings.MaxConcurrency
			if concurrency < 1 {
				concurrency = 5
			}

			// Determine if we need to fetch details
			fetchDetails := detail || fwVersionHas != "" || fwVersionNot != "" || netSpeedGT > -1 || netSpeedLT > -1

			// Results channel
			resultsChan := make(chan interface{}, len(targets))
			var wg sync.WaitGroup
			sem := make(chan struct{}, concurrency)

			// Start workers (shared between TUI and non-TUI)
			for _, s := range targets {
				wg.Add(1)
				go func(server config.LocalServerConfig) {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()

					stats := fetchServerStatus(server, serverOrder[server.URL], cfg, fetchDetails, &opts)
					resultsChan <- stats
				}(s)
			}

			go func() {
				wg.Wait()
				close(resultsChan)
			}()

			// ========== Collect results ==========
			var results []ServerStatusStats

			if output == "plain" || output == "json" {
				// Non-TUI: collect directly from channel, no BubbleTea
				processed := 0
				for raw := range resultsChan {
					processed++
					results = append(results, raw.(ServerStatusStats))
					if output == "plain" {
						fmt.Fprintf(os.Stderr, "\r进度: %d/%d", processed, len(targets))
					}
				}
				if output == "plain" {
					fmt.Fprintln(os.Stderr)
				}
			} else {
				// TUI mode: use BubbleTea progress bar
				totalDevicesStatus := 0
				prog := tea.NewProgram(tui.NewProgressModel(len(targets), resultsChan, func(v interface{}) string {
					stats := v.(ServerStatusStats)
					url := cleanURL(stats.ServerURL)
					if stats.Status == "Online" {
						totalDevicesStatus += stats.DeviceCount
						return fmt.Sprintf("✅ %s: %d 台设备 (总计: %d)", url, stats.DeviceCount, totalDevicesStatus)
					}
					return fmt.Sprintf("❌ %s: %s", url, stats.Status)
				}))

				finalModel, err := prog.Run()
				if err != nil {
					return fmt.Errorf("TUI运行错误: %v", err)
				}

				rawResults := finalModel.(tui.ProgressModel).GetResults()
				for _, raw := range rawResults {
					results = append(results, raw.(ServerStatusStats))
				}
			}

			// ========== Post-filter ==========
			results = filterStatusResults(results, &opts,
				bizOnlineGT, bizOnlineLT, ipCountGT, ipCountLT,
				uuidCountGT, uuidCountLT, snGT, snLT,
				authFailed, clusterContains, clusterNotContains,
				fwVersionHas, fwVersionNot, netSpeedGT, netSpeedLT)

			sort.Slice(results, func(i, j int) bool {
				return results[i].OrderIndex < results[j].OrderIndex
			})

			if len(results) == 0 {
				if output == "json" {
					fmt.Println("{\"summary\":{\"total_servers\":0},\"servers\":[]}")
					return nil
				}
				fmt.Println("没有找到匹配的服务器或设备。")
				return nil
			}

			// ========== Output ==========
			switch output {
			case "json":
				printStatusJSON(results)
			case "plain":
				printStatusPlain(results, detail)
			default:
				printStatusTUI(results, detail)
			}

			return nil
		},
	}

	AddCommonFlags(cmd, &opts)
	cmd.Flags().BoolVar(&detail, "detail", false, "显示详细授权信息 (SN, 集控地址, 授权名称)")

	// Advanced Filter Flags
	cmd.Flags().IntVar(&bizOnlineGT, "biz-online-gt", -1, "筛选业务在线数大于指定值的服务器")
	cmd.Flags().IntVar(&bizOnlineLT, "biz-online-lt", -1, "筛选业务在线数小于指定值的服务器")
	cmd.Flags().IntVar(&ipCountGT, "ip-count-gt", -1, "筛选IP数大于指定值的服务器")
	cmd.Flags().IntVar(&ipCountLT, "ip-count-lt", -1, "筛选IP数小于指定值的服务器")
	cmd.Flags().IntVar(&uuidCountGT, "uuid-count-gt", -1, "筛选UUID数大于指定值的服务器")
	cmd.Flags().IntVar(&uuidCountLT, "uuid-count-lt", -1, "筛选UUID数小于指定值的服务器")
	cmd.Flags().StringVar(&snGT, "sn-gt", "", "筛选序列号大于指定值的服务器")
	cmd.Flags().StringVar(&snLT, "sn-lt", "", "筛选序列号小于指定值的服务器")
	cmd.Flags().BoolVar(&authFailed, "auth-failed", false, "筛选授权状态非成功的服务器")
	cmd.Flags().StringVar(&clusterContains, "cluster-contains", "", "筛选集控平台地址包含指定字符串的服务器")
	cmd.Flags().StringVar(&clusterNotContains, "cluster-not-contains", "", "筛选集控平台地址不包含指定字符串的服务器")
	cmd.Flags().StringVar(&fwVersionHas, "fw-has", "", "筛选固件版本包含指定字符串的服务器")
	cmd.Flags().StringVar(&fwVersionNot, "fw-not", "", "筛选固件版本不包含指定字符串的服务器")
	cmd.Flags().Float64Var(&netSpeedGT, "speed-gt", -1, "筛选网络速率大于指定值(Mbps)的服务器")
	cmd.Flags().Float64Var(&netSpeedLT, "speed-lt", -1, "筛选网络速率小于指定值(Mbps)的服务器")

	return cmd
}

// fetchServerStatus 获取单台服务器状态（从原 goroutine 逻辑提取）
func fetchServerStatus(server config.LocalServerConfig, orderIndex int, cfg *config.Config, fetchDetails bool, opts *CommonFlags) ServerStatusStats {
	stats := ServerStatusStats{
		ServerURL:     server.URL,
		Status:        "Offline",
		LicenseStatus: "Unknown",
		OrderIndex:    orderIndex,
	}

	// 1. Check License (HTTP)
	apiClient := httpclient.NewClient(server.URL, server.Token)
	lic, err := apiClient.GetLicense()
	if err == nil {
		if lic.StatusTxt == "成功" && lic.C == "" {
			logger.Warnf("[%s] Authorization successful but Control Platform missing. Re-submitting...", server.URL)
			if reauthErr := apiClient.Reauthorize(lic.Sn); reauthErr == nil {
				if newLic, refreshErr := apiClient.GetLicense(); refreshErr == nil {
					lic = newLic
				}
			}
		}
		fillLicenseStats(&stats, lic)
		stats.Status = "Online"
	} else {
		// Re-login
		logger.Infof("[%s] License check failed, attempting re-login...", server.URL)
		newToken, loginErr := apiClient.Login(server.Username, server.Password)
		if loginErr == nil {
			server.Token = newToken
			server.LastLoginTime = time.Now().Format(time.RFC3339)
			server.LastLoginError = ""
			config.UpdateServer(cfg, server)

			apiClient.Token = newToken
			lic, err = apiClient.GetLicense()
			if err == nil {
				if lic.StatusTxt == "成功" && lic.C == "" {
					if reauthErr := apiClient.Reauthorize(lic.Sn); reauthErr == nil {
						if newLic, refreshErr := apiClient.GetLicense(); refreshErr == nil {
							lic = newLic
						}
					}
				}
				fillLicenseStats(&stats, lic)
				stats.Status = "Online"
			} else {
				stats.Status = "AuthFail"
			}
		} else {
			stats.Status = "AuthFail"
			server.LastLoginError = loginErr.Error()
			config.UpdateServer(cfg, server)
		}
	}

	// 2. Fetch Devices (WS)
	if stats.Status == "Online" {
		wsClient := wsclient.NewClient(server.URL, server.Token)
		wsClient.Timeout = time.Duration(config.GlobalSettings.ConnectTimeout) * time.Second
		if err := wsClient.Connect(); err == nil {
			deviceAPI := api.NewDeviceAPI(wsClient, server.URL, server.Token)

			if fetchDetails {
				if version, err := deviceAPI.GetSystemVersion(); err == nil {
					stats.FirmwareVersion = version.Version
				}
				if networkInfo, err := deviceAPI.GetNetworkInfo(); err == nil && networkInfo.Speed != nil && networkInfo.Speed.Double != nil {
					speed := *networkInfo.Speed.Double
					stats.NetworkSpeedVal = speed
					if speed > 0 {
						stats.NetworkSpeed = fmt.Sprintf("%.1f Mbps", speed)
					}
				}
			}

			devices, _ := deviceAPI.FetchDeviceList()
			onlineStatuses, err := deviceAPI.FetchOnlineStatus()
			if err == nil {
				statusMap := make(map[int]model.OnlineStatus)
				for _, s := range onlineStatuses {
					s.Parse()
					statusMap[s.Seat] = s
				}

				for _, d := range devices {
					if opts.UUID != "" && !strings.Contains(d.UUID, opts.UUID) {
						continue
					}
					if opts.Seat > -1 && d.Seat != opts.Seat {
						continue
					}

					var isOnline, isBizOnline, isADB, isUSB, hasIP bool
					if s, ok := statusMap[d.Seat]; ok {
						isOnline = true
						isBizOnline = s.IsManagementOnline
						isADB = s.IsADBEnabled
						isUSB = s.IsUSBMode
						hasIP = s.IP != ""
					}

					if opts.FilterOnline != "" && isOnline != (opts.FilterOnline == "true") {
						continue
					}
					if opts.FilterADB != "" && isADB != (opts.FilterADB == "true") {
						continue
					}
					if opts.FilterUSB != "" && isUSB != (opts.FilterUSB == "true") {
						continue
					}
					if opts.FilterHasIP != "" && hasIP != (opts.FilterHasIP == "true") {
						continue
					}

					stats.DeviceCount++
					if d.UUID != "" {
						stats.UUIDCount++
					}
					if isOnline {
						if isBizOnline && hasIP {
							stats.BizOnlineCount++
						}
						if hasIP {
							stats.IPCount++
						}
						if isADB {
							stats.ADBCount++
						}
						if isUSB {
							stats.USBCount++
						} else {
							stats.OTGCount++
						}
					}
				}
			}
			wsClient.Close()
		}
	}

	return stats
}

// fillLicenseStats 将授权信息填充到 stats
func fillLicenseStats(stats *ServerStatusStats, lic *model.LicenseData) {
	if lic.StatusTxt != "" {
		stats.LicenseStatus = lic.StatusTxt
	} else {
		stats.LicenseStatus = "Unknown"
	}
	if lic.Sn != "" {
		stats.SN = lic.Sn
	}
	if lic.C != "" {
		stats.ControlAddr = lic.C
	}
	if lic.N != "" {
		stats.LicenseName = lic.N
	}
}

// filterStatusResults 过滤服务器状态结果
func filterStatusResults(results []ServerStatusStats, opts *CommonFlags,
	bizOnlineGT, bizOnlineLT, ipCountGT, ipCountLT,
	uuidCountGT, uuidCountLT int,
	snGT, snLT string,
	authFailed bool,
	clusterContains, clusterNotContains,
	fwVersionHas, fwVersionNot string,
	netSpeedGT, netSpeedLT float64,
) []ServerStatusStats {
	var filtered []ServerStatusStats
	for _, r := range results {
		keep := true

		if opts.AuthorizedOnly == "true" && r.LicenseStatus != "成功" {
			keep = false
		}
		if opts.AuthorizedOnly == "false" && r.LicenseStatus == "成功" {
			keep = false
		}
		if authFailed && r.LicenseStatus == "成功" {
			keep = false
		}
		if bizOnlineGT > -1 && r.BizOnlineCount <= bizOnlineGT {
			keep = false
		}
		if bizOnlineLT > -1 && r.BizOnlineCount >= bizOnlineLT {
			keep = false
		}
		if ipCountGT > -1 && r.IPCount <= ipCountGT {
			keep = false
		}
		if ipCountLT > -1 && r.IPCount >= ipCountLT {
			keep = false
		}
		if uuidCountGT > -1 && r.UUIDCount <= uuidCountGT {
			keep = false
		}
		if uuidCountLT > -1 && r.UUIDCount >= uuidCountLT {
			keep = false
		}
		if snGT != "" && r.SN <= snGT {
			keep = false
		}
		if snLT != "" && r.SN >= snLT {
			keep = false
		}
		if clusterContains != "" && !strings.Contains(r.ControlAddr, clusterContains) {
			keep = false
		}
		if clusterNotContains != "" && strings.Contains(r.ControlAddr, clusterNotContains) {
			keep = false
		}
		if fwVersionHas != "" && !strings.Contains(r.FirmwareVersion, fwVersionHas) {
			keep = false
		}
		if fwVersionNot != "" && strings.Contains(r.FirmwareVersion, fwVersionNot) {
			keep = false
		}
		if netSpeedGT > -1 && r.NetworkSpeedVal <= netSpeedGT {
			keep = false
		}
		if netSpeedLT > -1 && r.NetworkSpeedVal >= netSpeedLT {
			keep = false
		}

		hasDeviceFilter := opts.UUID != "" || opts.Seat > -1 || opts.FilterOnline != "" ||
			opts.FilterADB != "" || opts.FilterUSB != "" || opts.FilterHasIP != ""
		if hasDeviceFilter && r.DeviceCount == 0 {
			keep = false
		}

		if keep {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// printStatusTUI 使用 lipgloss 渲染表格（原 TUI 表格输出）
func printStatusTUI(results []ServerStatusStats, detail bool) {
	var (
		headerStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				Align(lipgloss.Center)

		cellStyle = lipgloss.NewStyle().
				Align(lipgloss.Center)

		statusOnlineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
		statusErrorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

		numGoodStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
		numBadStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		numNeutralStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

		headers []string
		widths  []int
	)

	headers = []string{"服务器地址", "状态"}
	widths = []int{24, 10}

	if detail {
		headers = append(headers, "固件版本", "网络速率")
		widths = append(widths, 12, 12)
	}

	headers = append(headers, "设备数", "业务在线", "IP", "序列号", "ADB", "模式(OTG/USB)")
	widths = append(widths, 8, 12, 12, 14, 14, 14)

	if detail {
		headers = append(headers, "授权(状态/SN/集控/名称)")
		widths = append(widths, 90)
	} else {
		headers = append(headers, "授权")
		widths = append(widths, 30)
	}

	renderStats := func(good, bad int, isMode bool) string {
		sGood := fmt.Sprintf("%d", good)
		sBad := fmt.Sprintf("%d", bad)
		stGood := numGoodStyle
		stBad := numNeutralStyle
		if bad > 0 && !isMode {
			stBad = numBadStyle
		}
		if isMode {
			stBad = numGoodStyle
		}
		return fmt.Sprintf("%s/%s", stGood.Render(sGood), stBad.Render(sBad))
	}

	var headerRow string
	for i, h := range headers {
		headerRow = lipgloss.JoinHorizontal(lipgloss.Top, headerRow, headerStyle.Width(widths[i]).Render(h))
	}
	fmt.Println(headerRow)
	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("-", lipgloss.Width(headerRow))))

	var (
		totalDevice, totalBizOnline, totalIP, totalUUID, totalADB, totalUSB, totalOTG int
		totalNormalSpeed, totalLowSpeed                                                int
	)

	for _, r := range results {
		totalDevice += r.DeviceCount
		totalBizOnline += r.BizOnlineCount
		totalIP += r.IPCount
		totalUUID += r.UUIDCount
		totalADB += r.ADBCount
		totalUSB += r.USBCount
		totalOTG += r.OTGCount

		displayURL := cleanURL(r.ServerURL)

		stStatus := statusOnlineStyle
		if r.Status != "Online" {
			stStatus = statusErrorStyle
		}

		stSpeed := cellStyle
		if r.NetworkSpeed != "" {
			if r.NetworkSpeedVal < 1000 {
				stSpeed = statusErrorStyle
				totalLowSpeed++
			} else {
				totalNormalSpeed++
			}
		}

		stLic := statusOnlineStyle
		if r.LicenseStatus != "成功" {
			stLic = statusErrorStyle
		}

		authInfo := r.LicenseStatus
		if detail {
			var details []string
			if r.SN != "" {
				details = append(details, fmt.Sprintf("SN:%s", r.SN))
			}
			if r.ControlAddr != "" {
				details = append(details, fmt.Sprintf("C:%s", r.ControlAddr))
			}
			if r.LicenseName != "" {
				details = append(details, fmt.Sprintf("N:%s", r.LicenseName))
			}
			if len(details) > 0 {
				authInfo = fmt.Sprintf("%s | %s", r.LicenseStatus, strings.Join(details, " | "))
			}
		}

		row := []string{
			cellStyle.Width(widths[0]).Render(displayURL),
			stStatus.Width(widths[1]).Render(r.Status),
		}

		idx := 2
		if detail {
			row = append(row, cellStyle.Width(widths[idx]).Render(r.FirmwareVersion))
			idx++
			row = append(row, stSpeed.Width(widths[idx]).Render(r.NetworkSpeed))
			idx++
		}

		row = append(row,
			cellStyle.Width(widths[idx]).Render(fmt.Sprintf("%d", r.DeviceCount)),
			cellStyle.Width(widths[idx+1]).Render(renderStats(r.BizOnlineCount, r.DeviceCount-r.BizOnlineCount, false)),
			cellStyle.Width(widths[idx+2]).Render(renderStats(r.IPCount, r.DeviceCount-r.IPCount, false)),
			cellStyle.Width(widths[idx+3]).Render(renderStats(r.UUIDCount, r.DeviceCount-r.UUIDCount, false)),
			cellStyle.Width(widths[idx+4]).Render(renderStats(r.ADBCount, r.DeviceCount-r.ADBCount, false)),
			cellStyle.Width(widths[idx+5]).Render(renderStats(r.OTGCount, r.USBCount, true)),
			stLic.Width(widths[idx+6]).Render(authInfo),
		)

		var rowStr string
		for _, cell := range row {
			rowStr = lipgloss.JoinHorizontal(lipgloss.Top, rowStr, cell)
		}
		fmt.Println(rowStr)
	}

	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("-", lipgloss.Width(headerRow))))

	totalRow := []string{
		cellStyle.Width(widths[0]).Render(fmt.Sprintf("总计 (%d 台服务器)", len(results))),
		cellStyle.Width(widths[1]).Render("-"),
	}

	idx := 2
	if detail {
		totalRow = append(totalRow, cellStyle.Width(widths[idx]).Render("-"))
		idx++
		totalRow = append(totalRow, cellStyle.Width(widths[idx]).Render(renderStats(totalNormalSpeed, totalLowSpeed, false)))
		idx++
	}

	totalRow = append(totalRow,
		cellStyle.Width(widths[idx]).Render(fmt.Sprintf("%d", totalDevice)),
		cellStyle.Width(widths[idx+1]).Render(renderStats(totalBizOnline, totalDevice-totalBizOnline, false)),
		cellStyle.Width(widths[idx+2]).Render(renderStats(totalIP, totalDevice-totalIP, false)),
		cellStyle.Width(widths[idx+3]).Render(renderStats(totalUUID, totalDevice-totalUUID, false)),
		cellStyle.Width(widths[idx+4]).Render(renderStats(totalADB, totalDevice-totalADB, false)),
		cellStyle.Width(widths[idx+5]).Render(renderStats(totalOTG, totalUSB, true)),
		cellStyle.Width(widths[idx+6]).Render("-"),
	)
	var totalStr string
	for _, cell := range totalRow {
		totalStr = lipgloss.JoinHorizontal(lipgloss.Top, totalStr, cell)
	}
	fmt.Println(totalStr)

	fmt.Println(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(strings.Repeat("-", lipgloss.Width(headerRow))))
	summary := fmt.Sprintf("总计: %d 台 | 业务在线: %d | IP: %d | UUID: %d | ADB开启: %d | USB: %d | OTG: %d",
		totalDevice, totalBizOnline, totalIP, totalUUID, totalADB, totalUSB, totalOTG)
	fmt.Println(lipgloss.NewStyle().Bold(true).Render(summary))
}

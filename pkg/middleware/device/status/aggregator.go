package status

import (
	"context"
	"fmt"
	httpclient "jpy-cli/pkg/client/http"
	"jpy-cli/pkg/config"
	"jpy-cli/pkg/middleware/connector"
	"jpy-cli/pkg/middleware/device/api"
	"jpy-cli/pkg/middleware/device/selector"
	"strings"
	"sync"
	"time"
)

type ServerStatusStats struct {
	ServerURL       string  `json:"server_url"`
	Status          string  `json:"status"`
	LicenseStatus   string  `json:"license_status"`
	SN              string  `json:"sn"`
	ControlAddr     string  `json:"control_addr"`
	LicenseName     string  `json:"license_name"`
	FirmwareVersion string  `json:"firmware_version"`
	NetworkSpeed    string  `json:"network_speed"`
	NetworkSpeedVal float64 `json:"network_speed_val"`
	DeviceCount     int     `json:"device_count"`
	BizOnlineCount  int     `json:"biz_online_count"`
	IPCount         int     `json:"ip_count"`
	UUIDCount       int     `json:"uuid_count"`
	ADBCount        int     `json:"adb_count"`
	USBCount        int     `json:"usb_count"`
	OTGCount        int     `json:"otg_count"`
	Error           string  `json:"error,omitempty"`
	Disabled        bool    `json:"disabled"`
	LastLoginError  string  `json:"last_login_error,omitempty"`
	OrderIndex      int     `json:"order_index"`
}

type StatusFilters struct {
	Group         string
	ServerPattern string

	// Server Filters
	AuthorizedOnly     *bool
	ShowDisabled       bool
	DisabledOnly       bool
	AuthFailed         bool
	ClusterContains    string
	ClusterNotContains string
	FwVersionHas       string
	FwVersionNot       string
	NetSpeedGT         float64
	NetSpeedLT         float64

	// Device Count Filters
	BizOnlineGT int
	BizOnlineLT int
	IPCountGT   int
	IPCountLT   int
	UUIDCountGT int
	UUIDCountLT int

	// Device Property Filters (for counting)
	UUID         string
	Seat         int
	FilterOnline *bool
	FilterADB    *bool
	FilterUSB    *bool
	FilterHasIP  *bool

	// Misc
	Detail bool
	SNGT   string
	SNLT   string
}

func GetServerStatusStats(ctx context.Context, cfg *config.Config, filters StatusFilters, progressCb func(int, int, string)) ([]ServerStatusStats, error) {
	// Determine group
	targetGroup := filters.Group
	if targetGroup == "" {
		targetGroup = cfg.ActiveGroup
	}
	if targetGroup == "" {
		targetGroup = "default"
	}

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
		// Skip soft-deleted servers unless ShowDisabled is true
		if s.Disabled && !filters.ShowDisabled {
			continue
		}
		// If DisabledOnly is set, skip enabled servers
		if !s.Disabled && filters.DisabledOnly {
			continue
		}
		if selector.MatchServerPattern(s.URL, filters.ServerPattern) {
			targets = append(targets, s)
		}
	}

	if len(targets) == 0 {
		return nil, nil
	}

	concurrency := config.GlobalSettings.MaxConcurrency
	if concurrency < 1 {
		concurrency = 5
	}

	// Determine if we need to fetch details
	fetchDetails := filters.Detail || filters.FwVersionHas != "" || filters.FwVersionNot != "" || filters.NetSpeedGT > -1 || filters.NetSpeedLT > -1

	// Results channel
	resultsChan := make(chan ServerStatusStats, len(targets))
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	// Start workers
	for _, s := range targets {
		// Optimization: Handle disabled servers without spawning goroutines
		if s.Disabled {
			stats := ServerStatusStats{
				ServerURL:      s.URL,
				Status:         "已移除",
				LicenseStatus:  "Unknown",
				Disabled:       true,
				LastLoginError: s.LastLoginError,
				OrderIndex:     serverOrder[s.URL],
			}
			// Non-blocking send if possible, but resultsChan is buffered to len(targets) so it should be fine
			resultsChan <- stats
			continue
		}

		wg.Add(1)
		go func(server config.LocalServerConfig) {
			defer wg.Done()

			// Acquire semaphore with context check
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			// Check context
			select {
			case <-ctx.Done():
				return
			default:
			}

			stats := ServerStatusStats{
				ServerURL:      server.URL,
				Status:         "Offline",
				LicenseStatus:  "Unknown",
				Disabled:       server.Disabled,
				LastLoginError: server.LastLoginError,
				OrderIndex:     serverOrder[server.URL],
			}

			// 1. Check License (HTTP)
			apiClient := httpclient.NewClient(server.URL, server.Token)
			apiClient.SetTimeout(1 * time.Second) // Force 1s timeout for status check
			lic, err := apiClient.GetLicense()
			if err == nil {
				// Re-authorize logic
				if lic.StatusTxt == "成功" && lic.C == "" {
					if reauthErr := apiClient.Reauthorize(lic.Sn); reauthErr == nil {
						if newLic, refreshErr := apiClient.GetLicense(); refreshErr == nil {
							lic = newLic
						}
					}
				}

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
				stats.Status = "Online"
			} else {
				// Attempt re-login
				newToken, loginErr := apiClient.Login(server.Username, server.Password)
				if loginErr == nil {
					// Update server config
					server.Token = newToken
					server.LastLoginTime = time.Now().Format(time.RFC3339)
					server.LastLoginError = ""
					config.UpdateServer(cfg, server)

					// Retry License Check
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
						stats.Status = "Online"
					} else {
						stats.Status = "Offline"
						stats.Error = fmt.Sprintf("License check failed: %v", err)
					}
				} else {
					stats.Status = "Offline"
					stats.Error = fmt.Sprintf("Login failed: %v", loginErr)

					// Update LastLoginError
					server.LastLoginError = stats.Error
					config.UpdateServer(cfg, server)
				}
			}

			// Check context before heavy WebSocket op
			select {
			case <-ctx.Done():
				return
			default:
			}

			// 2. Fetch Devices (WS) if Online
			if stats.Status == "Online" {
				connector := connector.NewConnectorService(cfg)
				ws, err := connector.Connect(server)
				if err == nil {
					defer ws.Close()
					deviceAPI := api.NewDeviceAPI(ws, server.URL, server.Token)

					// Get counts
					statuses, err := deviceAPI.FetchOnlineStatus()
					if err == nil {
						stats.DeviceCount = len(statuses)
						for _, s := range statuses {
							s.Parse()
							if s.IsBusinessOnline && s.IsManagementOnline {
								stats.BizOnlineCount++
							}
							if s.IP != "" {
								stats.IPCount++
							}
							if s.IsADBEnabled {
								stats.ADBCount++
							}
							if s.IsUSBMode {
								stats.USBCount++
							} else {
								stats.OTGCount++
							}
						}

						// Fetch UUIDs if we need UUID count or detail
						// This requires FetchDeviceList which is heavier
						if filters.Detail || filters.UUIDCountGT > -1 || filters.UUIDCountLT > -1 {
							devices, err := deviceAPI.FetchDeviceList()
							if err == nil {
								uuidCount := 0
								for _, d := range devices {
									if d.UUID != "" {
										uuidCount++
									}
								}
								stats.UUIDCount = uuidCount
							}
						}
					}

					// 3. Fetch Detail (Firmware, Network Speed) if requested
					if fetchDetails {
						sysVer, err := deviceAPI.GetSystemVersion()
						if err == nil && sysVer != nil {
							stats.FirmwareVersion = sysVer.Version
						}

						netInfo, err := deviceAPI.GetNetworkInfo()
						if err == nil && netInfo != nil {
							if netInfo.Speed != nil && netInfo.Speed.Double != nil {
								val := *netInfo.Speed.Double
								stats.NetworkSpeedVal = val
								stats.NetworkSpeed = fmt.Sprintf("%.2f MB/s", val)
							}
						}
					}
				}
			}

			select {
			case resultsChan <- stats:
			case <-ctx.Done():
			}
		}(s)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	var results []ServerStatusStats
	processed := 0
	total := len(targets)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case r, ok := <-resultsChan:
			if !ok {
				goto Done
			}
			processed++
			if progressCb != nil {
				cleanURL := strings.TrimPrefix(r.ServerURL, "https://")
				cleanURL = strings.TrimPrefix(cleanURL, "http://")
				msg := fmt.Sprintf("[%d/%d] Checked %s: %s", processed, total, cleanURL, r.Status)
				progressCb(processed, total, msg)
			}
			results = append(results, r)
		}
	}
Done:

	// Apply Server Filters (Post-Processing)
	var filteredResults []ServerStatusStats
	for _, r := range results {
		keep := true

		if filters.AuthorizedOnly != nil {
			reqAuth := *filters.AuthorizedOnly
			isAuth := (r.LicenseStatus == "成功")
			if isAuth != reqAuth {
				keep = false
			}
		}
		if filters.AuthFailed && r.LicenseStatus == "成功" {
			keep = false
		}
		if filters.BizOnlineGT > -1 && r.BizOnlineCount <= filters.BizOnlineGT {
			keep = false
		}
		if filters.BizOnlineLT > -1 && r.BizOnlineCount >= filters.BizOnlineLT {
			keep = false
		}
		if filters.IPCountGT > -1 && r.IPCount <= filters.IPCountGT {
			keep = false
		}
		if filters.IPCountLT > -1 && r.IPCount >= filters.IPCountLT {
			keep = false
		}
		if filters.UUIDCountGT > -1 && r.UUIDCount <= filters.UUIDCountGT {
			keep = false
		}
		if filters.UUIDCountLT > -1 && r.UUIDCount >= filters.UUIDCountLT {
			keep = false
		}
		if filters.SNGT != "" && r.SN <= filters.SNGT {
			keep = false
		}
		if filters.SNLT != "" && r.SN >= filters.SNLT {
			keep = false
		}
		if filters.ClusterContains != "" && !strings.Contains(r.ControlAddr, filters.ClusterContains) {
			keep = false
		}
		if filters.ClusterNotContains != "" && strings.Contains(r.ControlAddr, filters.ClusterNotContains) {
			keep = false
		}
		if filters.FwVersionHas != "" && !strings.Contains(r.FirmwareVersion, filters.FwVersionHas) {
			keep = false
		}
		if filters.FwVersionNot != "" && strings.Contains(r.FirmwareVersion, filters.FwVersionNot) {
			keep = false
		}
		if filters.NetSpeedGT > -1 && r.NetworkSpeedVal <= filters.NetSpeedGT {
			keep = false
		}
		if filters.NetSpeedLT > -1 && r.NetworkSpeedVal >= filters.NetSpeedLT {
			keep = false
		}

		hasDeviceFilter := filters.UUID != "" || filters.Seat > -1 || filters.FilterOnline != nil ||
			filters.FilterADB != nil || filters.FilterUSB != nil || filters.FilterHasIP != nil

		if hasDeviceFilter && r.DeviceCount == 0 {
			keep = false
		}

		if keep {
			filteredResults = append(filteredResults, r)
		}
	}

	return filteredResults, nil
}

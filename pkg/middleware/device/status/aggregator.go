package status

import (
	"context"
	"fmt"
	httpclient "jpy-cli/pkg/client/http"
	"jpy-cli/pkg/middleware/connector"
	"jpy-cli/pkg/middleware/device/api"
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
	OrderIndex      int     `json:"order_index"`
}

type StatusFilters struct {
	Detail bool
}

// GetServerStatusStats 获取服务器状态统计（无状态，直传凭证）
func GetServerStatusStats(ctx context.Context, servers []connector.ServerInfo, filters StatusFilters, progressCb func(int, int, string)) ([]ServerStatusStats, error) {
	if len(servers) == 0 {
		return nil, nil
	}

	concurrency := 5
	resultsChan := make(chan ServerStatusStats, len(servers))
	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	for i, s := range servers {
		wg.Add(1)
		go func(server connector.ServerInfo, idx int) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			stats := ServerStatusStats{
				ServerURL:  server.URL,
				Status:     "Offline",
				OrderIndex: idx,
			}

			// 1. Check License (HTTP)
			apiClient := httpclient.NewClient(server.URL, server.Token)
			apiClient.SetTimeout(1 * time.Second)
			lic, err := apiClient.GetLicense()
			if err != nil {
				// 尝试重新登录
				newToken, loginErr := apiClient.Login(server.Username, server.Password)
				if loginErr != nil {
					stats.Error = fmt.Sprintf("Login failed: %v", loginErr)
					resultsChan <- stats
					return
				}
				server.Token = newToken
				apiClient.Token = newToken
				lic, err = apiClient.GetLicense()
				if err != nil {
					stats.Error = fmt.Sprintf("License check failed: %v", err)
					resultsChan <- stats
					return
				}
			}

			stats.Status = "Online"
			if lic.StatusTxt != "" {
				stats.LicenseStatus = lic.StatusTxt
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

			// 2. Fetch device stats via WS
			ws, err := connector.Connect(server)
			if err == nil {
				defer ws.Close()
				deviceAPI := api.NewDeviceAPI(ws, server.URL, server.Token)

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
				}

				if filters.Detail {
					devices, err := deviceAPI.FetchDeviceList()
					if err == nil {
						for _, d := range devices {
							if d.UUID != "" {
								stats.UUIDCount++
							}
						}
					}

					sysVer, err := deviceAPI.GetSystemVersion()
					if err == nil && sysVer != nil {
						stats.FirmwareVersion = sysVer.Version
					}
				}

				// 网速默认获取
				netInfo, err := deviceAPI.GetNetworkInfo()
				if err == nil && netInfo != nil {
					if netInfo.Speed != nil && netInfo.Speed.Double != nil {
						val := *netInfo.Speed.Double
						stats.NetworkSpeedVal = val
						stats.NetworkSpeed = fmt.Sprintf("%.2f MB/s", val)
					}
				}
			}

			select {
			case resultsChan <- stats:
			case <-ctx.Done():
			}
		}(s, i)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	var results []ServerStatusStats
	processed := 0
	total := len(servers)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case r, ok := <-resultsChan:
			if !ok {
				return results, nil
			}
			processed++
			if progressCb != nil {
				cleanURL := strings.TrimPrefix(r.ServerURL, "https://")
				cleanURL = strings.TrimPrefix(cleanURL, "http://")
				progressCb(processed, total, fmt.Sprintf("[%d/%d] %s: %s", processed, total, cleanURL, r.Status))
			}
			results = append(results, r)
		}
	}
}

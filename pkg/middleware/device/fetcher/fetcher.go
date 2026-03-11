package fetcher

import (
	"context"
	"fmt"
	"jpy-cli/pkg/logger"
	"jpy-cli/pkg/middleware/connector"
	"jpy-cli/pkg/middleware/device/api"
	"jpy-cli/pkg/middleware/model"
	"sync"
)

// ServerResult contains the result of fetching devices from a single server
type ServerResult struct {
	ServerURL  string
	Devices    []model.DeviceListItem
	Statuses   []model.OnlineStatus
	Error      error
	OrderIndex int
}

// FetchDevices concurrently fetches devices from servers.
// 接受 connector.ServerInfo 切片，不依赖配置文件。
func FetchDevices(ctx context.Context, servers []connector.ServerInfo) (chan interface{}, int) {
	concurrency := 5

	resultsChan := make(chan interface{}, len(servers))

	serverOrder := make(map[string]int)
	for i, s := range servers {
		serverOrder[s.URL] = i
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, concurrency)

	for _, s := range servers {
		wg.Add(1)
		go func(server connector.ServerInfo) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-sem }()

			select {
			case <-ctx.Done():
				return
			default:
			}

			res := ServerResult{
				ServerURL:  server.URL,
				OrderIndex: serverOrder[server.URL],
			}

			ws, err := connector.Connect(server)
			if err != nil {
				res.Error = err
				select {
				case resultsChan <- res:
				case <-ctx.Done():
				}
				return
			}
			defer ws.Close()

			deviceAPI := api.NewDeviceAPI(ws, server.URL, server.Token)
			devices, err := deviceAPI.FetchDeviceList()
			if err != nil {
				res.Error = fmt.Errorf("获取设备列表失败: %v", err)
				select {
				case resultsChan <- res:
				case <-ctx.Done():
				}
				return
			}
			res.Devices = devices

			select {
			case <-ctx.Done():
				return
			default:
			}

			statuses, err := deviceAPI.FetchOnlineStatus()
			if err != nil {
				logger.Warnf("Fetch online status failed for %s: %v", server.URL, err)
			} else {
				res.Statuses = statuses
			}

			select {
			case resultsChan <- res:
			case <-ctx.Done():
			}
		}(s)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	return resultsChan, len(servers)
}

// ProcessResults converts raw results into a flat list of DeviceInfo
func ProcessResults(rawResults []interface{}) ([]model.DeviceInfo, int) {
	var allDevices []model.DeviceInfo
	var errorCount int

	for _, raw := range rawResults {
		res := raw.(ServerResult)
		if res.Error != nil {
			errorCount++
			logger.Warnf("Server %s failed: %v", res.ServerURL, res.Error)
			continue
		}

		statusMap := make(map[int]model.OnlineStatus)
		for _, s := range res.Statuses {
			statusMap[s.Seat] = s
		}

		for _, d := range res.Devices {
			androidVer := ""
			if d.AndroidVersion != nil {
				androidVer = *d.AndroidVersion
			}

			info := model.DeviceInfo{
				ServerURL:   res.ServerURL,
				Seat:        d.Seat,
				UUID:        d.UUID,
				Model:       d.Model,
				Android:     androidVer,
				IsOnline:    false,
				ServerIndex: res.OrderIndex,
			}

			if s, ok := statusMap[d.Seat]; ok {
				s.Parse()
				if s.IsBusinessOnline && s.IsManagementOnline {
					info.IsOnline = true
				}
				if s.IP != "" {
					info.IP = s.IP
				}
				if s.IsManagementOnline {
					info.BizOnline = true
				}
				if s.IsADBEnabled {
					info.ADBEnabled = true
				}
				if s.IsUSBMode {
					info.USBMode = true
				}
			}
			allDevices = append(allDevices, info)
		}
	}
	return allDevices, errorCount
}

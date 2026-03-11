package controller

import (
	"fmt"
	"jpy-cli/pkg/logger"
	"jpy-cli/pkg/middleware/connector"
	"jpy-cli/pkg/middleware/device/api"
	"jpy-cli/pkg/middleware/device/terminal"
	"jpy-cli/pkg/middleware/model"
	"sync"
	"sync/atomic"
)

// DeviceController handles device control operations.
type DeviceController struct {
	servers map[string]connector.ServerInfo // key: serverURL
}

// NewDeviceController 创建控制器，传入服务器凭证映射
func NewDeviceController(servers map[string]connector.ServerInfo) *DeviceController {
	return &DeviceController{servers: servers}
}

// NewSingleServerController 单服务器快捷构造
func NewSingleServerController(server connector.ServerInfo) *DeviceController {
	return &DeviceController{
		servers: map[string]connector.ServerInfo{server.URL: server},
	}
}

func (c *DeviceController) findServer(url string) (connector.ServerInfo, bool) {
	s, ok := c.servers[url]
	return s, ok
}

// BatchResult 表示单个设备的操作结果
type BatchResult struct {
	Server string
	Seat   int
	UUID   string
	OK     bool
	Error  string
}

// executeBatchCollect 静默批量执行，返回结构化结果
func (c *DeviceController) executeBatchCollect(devices []model.DeviceInfo, action func(seat int, api *api.DeviceAPI) error, progressCb func(int, int)) ([]BatchResult, error) {
	devicesByServer := make(map[string][]model.DeviceInfo)
	for _, d := range devices {
		devicesByServer[d.ServerURL] = append(devicesByServer[d.ServerURL], d)
	}

	var results []BatchResult
	totalDevices := len(devices)
	processedCount := 0

	for serverURL, serverDevices := range devicesByServer {
		server, found := c.findServer(serverURL)
		if !found {
			for _, d := range serverDevices {
				processedCount++
				results = append(results, BatchResult{
					Server: serverURL, Seat: d.Seat, UUID: d.UUID,
					OK: false, Error: "缺少服务器凭证",
				})
				if progressCb != nil {
					progressCb(processedCount, totalDevices)
				}
			}
			continue
		}

		ws, err := connector.ConnectGuard(server)
		if err != nil {
			for _, d := range serverDevices {
				processedCount++
				results = append(results, BatchResult{
					Server: serverURL, Seat: d.Seat, UUID: d.UUID,
					OK: false, Error: fmt.Sprintf("连接失败: %v", err),
				})
				if progressCb != nil {
					progressCb(processedCount, totalDevices)
				}
			}
			continue
		}

		deviceAPI := api.NewDeviceAPI(ws, server.URL, server.Token)

		for _, d := range serverDevices {
			processedCount++
			err := action(d.Seat, deviceAPI)
			r := BatchResult{Server: serverURL, Seat: d.Seat, UUID: d.UUID, OK: err == nil}
			if err != nil {
				r.Error = err.Error()
			}
			results = append(results, r)
			if progressCb != nil {
				progressCb(processedCount, totalDevices)
			}
		}

		ws.Close()
	}

	return results, nil
}

// executeTerminalBatchCollect 静默终端批量执行
func (c *DeviceController) executeTerminalBatchCollect(devices []model.DeviceInfo, action func(seat int, term *terminal.TerminalSession) error, progressCb func(int, int)) ([]BatchResult, error) {
	devicesByServer := make(map[string][]model.DeviceInfo)
	for _, d := range devices {
		devicesByServer[d.ServerURL] = append(devicesByServer[d.ServerURL], d)
	}

	var results []BatchResult
	totalDevices := len(devices)
	processedCount := 0

	for serverURL, serverDevices := range devicesByServer {
		server, found := c.findServer(serverURL)
		if !found {
			for _, d := range serverDevices {
				processedCount++
				results = append(results, BatchResult{
					Server: serverURL, Seat: d.Seat, UUID: d.UUID,
					OK: false, Error: "缺少服务器凭证",
				})
				if progressCb != nil {
					progressCb(processedCount, totalDevices)
				}
			}
			continue
		}

		for _, d := range serverDevices {
			processedCount++
			ws, err := connector.ConnectDeviceTerminal(server, int64(d.Seat))
			if err != nil {
				results = append(results, BatchResult{
					Server: serverURL, Seat: d.Seat, UUID: d.UUID,
					OK: false, Error: fmt.Sprintf("连接终端失败: %v", err),
				})
				if progressCb != nil {
					progressCb(processedCount, totalDevices)
				}
				continue
			}

			term := terminal.NewTerminalSession(ws, int64(d.Seat))
			err = func() error {
				defer term.Close()
				if err := term.Init(); err != nil {
					return fmt.Errorf("终端初始化失败: %v", err)
				}
				return action(d.Seat, term)
			}()

			r := BatchResult{Server: serverURL, Seat: d.Seat, UUID: d.UUID, OK: err == nil}
			if err != nil {
				r.Error = err.Error()
			}
			results = append(results, r)
			if progressCb != nil {
				progressCb(processedCount, totalDevices)
			}
		}
	}

	return results, nil
}

// RebootBatchCollect 静默批量重启
func (c *DeviceController) RebootBatchCollect(devices []model.DeviceInfo, progressCb func(int, int)) ([]BatchResult, error) {
	return c.executeBatchCollect(devices, func(seat int, api *api.DeviceAPI) error {
		return api.RebootDevice(seat)
	}, progressCb)
}

// SwitchUSBBatchCollect 静默批量切换USB模式
func (c *DeviceController) SwitchUSBBatchCollect(devices []model.DeviceInfo, otg bool, progressCb func(int, int)) ([]BatchResult, error) {
	return c.executeBatchCollect(devices, func(seat int, api *api.DeviceAPI) error {
		return api.SwitchUSBMode(seat, otg)
	}, progressCb)
}

// ControlADBBatchCollect 静默批量ADB控制
func (c *DeviceController) ControlADBBatchCollect(devices []model.DeviceInfo, enable bool, progressCb func(int, int)) ([]BatchResult, error) {
	if enable {
		return c.executeBatchCollect(devices, func(seat int, api *api.DeviceAPI) error {
			return api.ControlADB(seat, enable)
		}, progressCb)
	}
	return c.executeTerminalBatchCollect(devices, func(seat int, term *terminal.TerminalSession) error {
		if err := term.Exec("settings put global adb_enabled 0"); err != nil {
			return fmt.Errorf("发送关闭指令失败: %v", err)
		}
		return nil
	}, progressCb)
}

// RestartServiceBatch 批量重启中间件服务
func (c *DeviceController) RestartServiceBatch(devices []model.DeviceInfo, service string, actionCode int) error {
	uniqueServers := make(map[string]struct{})
	var targetServers []string
	for _, d := range devices {
		if _, exists := uniqueServers[d.ServerURL]; !exists {
			uniqueServers[d.ServerURL] = struct{}{}
			targetServers = append(targetServers, d.ServerURL)
		}
	}

	totalServers := len(targetServers)
	concurrency := 5

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)
	successCount := int32(0)
	failCount := int32(0)

	for _, serverURL := range targetServers {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			server, found := c.findServer(url)
			if !found {
				atomic.AddInt32(&failCount, 1)
				logger.Errorf("缺少服务器凭证: %s", url)
				return
			}

			deviceAPI := api.NewDeviceAPI(nil, server.URL, server.Token)
			if err := deviceAPI.RestartService(service, actionCode); err != nil {
				atomic.AddInt32(&failCount, 1)
				logger.Errorf("重启服务失败 %s: %v", url, err)
			} else {
				atomic.AddInt32(&successCount, 1)
			}
		}(serverURL)
	}

	wg.Wait()

	if failCount > 0 {
		return fmt.Errorf("批量重启完成: 成功 %d/%d, 失败 %d", successCount, totalServers, failCount)
	}
	return nil
}

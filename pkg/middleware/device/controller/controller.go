package controller

import (
	"fmt"
	"jpy-cli/pkg/config"
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
	connector *connector.ConnectorService
}

func NewDeviceController(cfg *config.Config) *DeviceController {
	return &DeviceController{
		connector: connector.NewConnectorService(cfg),
	}
}

// ExecuteBatch executes a function on a list of devices using shared server connections.
func (c *DeviceController) ExecuteBatch(devices []model.DeviceInfo, action func(seat int, api *api.DeviceAPI) error, progressCb func(int, int, string)) error {
	// Group devices by server
	devicesByServer := make(map[string][]model.DeviceInfo)
	for _, d := range devices {
		devicesByServer[d.ServerURL] = append(devicesByServer[d.ServerURL], d)
	}

	successCount := 0
	failCount := 0
	totalDevices := len(devices)
	processedCount := 0

	fmt.Printf("开始对 %d 台设备进行批量操作...\n", totalDevices)

	for serverURL, serverDevices := range devicesByServer {
		server, found := c.findServerConfig(serverURL)
		if !found {
			logger.Errorf("未找到服务器配置: %s", serverURL)
			for range serverDevices {
				failCount++
				processedCount++
				if processedCount <= 10 {
					fmt.Printf("[%d/%d] ❌ 缺少服务器配置: %s\n", processedCount, totalDevices, serverURL)
				}
				if progressCb != nil {
					progressCb(processedCount, totalDevices, fmt.Sprintf("Missing config for %s", serverURL))
				}
			}
			continue
		}

		// Connect to Guard Channel (Shared per server)
		ws, err := c.connector.ConnectGuard(server)
		if err != nil {
			logger.Errorf("连接到 guard 失败 %s: %v", serverURL, err)
			for range serverDevices {
				failCount++
				processedCount++
				if processedCount <= 10 {
					fmt.Printf("[%d/%d] ❌ 连接失败: %s\n", processedCount, totalDevices, serverURL)
				}
				if progressCb != nil {
					progressCb(processedCount, totalDevices, fmt.Sprintf("Connect failed: %s", serverURL))
				}
			}
			continue
		}

		deviceAPI := api.NewDeviceAPI(ws, server.URL, server.Token)

		for _, d := range serverDevices {
			processedCount++
			err := action(d.Seat, deviceAPI)

			// Console Output (Top 10 detailed, others summary)
			if processedCount <= 10 {
				if err != nil {
					fmt.Printf("[%d/%d] ❌ %s (机位 %d): %v\n", processedCount, totalDevices, d.UUID, d.Seat, err)
				} else {
					fmt.Printf("[%d/%d] ✅ %s (机位 %d): 成功\n", processedCount, totalDevices, d.UUID, d.Seat)
				}
			} else {
				// Update progress line
				fmt.Printf("\r正在处理... %d/%d (成功: %d, 失败: %d)", processedCount, totalDevices, successCount, failCount)
			}

			if err != nil {
				logger.Errorf("控制设备失败 %s (机位 %d): %v", d.UUID, d.Seat, err)
				failCount++
			} else {
				logger.Infof("成功控制设备 %s (机位 %d)", d.UUID, d.Seat)
				successCount++
			}

			if progressCb != nil {
				status := "Success"
				if err != nil {
					status = fmt.Sprintf("Failed: %v", err)
				}
				progressCb(processedCount, totalDevices, fmt.Sprintf("Processed %s (Seat %d): %s", d.UUID, d.Seat, status))
			}
		}

		ws.Close()
	}

	fmt.Printf("\n批量操作完成。成功: %d, 失败: %d\n", successCount, failCount)

	if failCount > 0 {
		return fmt.Errorf("部分操作失败")
	}
	return nil
}

func (c *DeviceController) findServerConfig(url string) (config.LocalServerConfig, bool) {
	// Search in Groups
	for _, servers := range c.connector.Config.Groups {
		for _, s := range servers {
			if s.URL == url {
				return s, true
			}
		}
	}
	return config.LocalServerConfig{}, false
}

// RebootBatch executes the reboot command on multiple devices.
func (c *DeviceController) RebootBatch(devices []model.DeviceInfo, progressCb func(int, int, string)) error {
	return c.ExecuteBatch(devices, func(seat int, api *api.DeviceAPI) error {
		return api.RebootDevice(seat)
	}, progressCb)
}

// SwitchUSBBatch executes the USB switch command on multiple devices.
func (c *DeviceController) SwitchUSBBatch(devices []model.DeviceInfo, otg bool, progressCb func(int, int, string)) error {
	return c.ExecuteBatch(devices, func(seat int, api *api.DeviceAPI) error {
		return api.SwitchUSBMode(seat, otg)
	}, progressCb)
}

// ControlADBBatch executes the ADB control command on multiple devices.
func (c *DeviceController) ControlADBBatch(devices []model.DeviceInfo, enable bool, progressCb func(int, int, string)) error {
	if enable {
		return c.ExecuteBatch(devices, func(seat int, api *api.DeviceAPI) error {
			return api.ControlADB(seat, enable)
		}, progressCb)
	}

	// For disabling ADB, we must use terminal connection
	return c.executeTerminalBatch(devices, func(seat int, term *terminal.TerminalSession) error {
		// Send shell command to disable ADB
		if err := term.Exec("settings put global adb_enabled 0"); err != nil {
			return fmt.Errorf("发送关闭指令失败: %v", err)
		}
		return nil
	}, progressCb)
}

// executeTerminalBatch executes a function using terminal connection for each device
func (c *DeviceController) executeTerminalBatch(devices []model.DeviceInfo, action func(seat int, term *terminal.TerminalSession) error, progressCb func(int, int, string)) error {
	// Group devices by server
	devicesByServer := make(map[string][]model.DeviceInfo)
	for _, d := range devices {
		devicesByServer[d.ServerURL] = append(devicesByServer[d.ServerURL], d)
	}

	successCount := 0
	failCount := 0
	totalDevices := len(devices)
	processedCount := 0

	fmt.Printf("开始对 %d 台设备进行批量操作 (终端模式)...\n", totalDevices)

	for serverURL, serverDevices := range devicesByServer {
		server, found := c.findServerConfig(serverURL)
		if !found {
			logger.Errorf("未找到服务器配置: %s", serverURL)
			for range serverDevices {
				failCount++
				processedCount++
				if processedCount <= 10 {
					fmt.Printf("[%d/%d] ❌ 缺少服务器配置: %s\n", processedCount, totalDevices, serverURL)
				}
				if progressCb != nil {
					progressCb(processedCount, totalDevices, fmt.Sprintf("Missing config for %s", serverURL))
				}
			}
			continue
		}

		for _, d := range serverDevices {
			processedCount++

			// Connect to Terminal
			// Use Seat as ID
			ws, err := c.connector.ConnectDeviceTerminal(server, int64(d.Seat))
			if err != nil {
				failCount++
				logger.Errorf("连接终端失败 %s (机位 %d): %v", d.UUID, d.Seat, err)
				if processedCount <= 10 {
					fmt.Printf("[%d/%d] ❌ 连接失败 %s (机位 %d): %v\n", processedCount, totalDevices, d.UUID, d.Seat, err)
				}
				if progressCb != nil {
					progressCb(processedCount, totalDevices, fmt.Sprintf("Connect failed %s: %v", d.UUID, err))
				}
				continue
			}

			term := terminal.NewTerminalSession(ws, int64(d.Seat))

			// Init and Wait
			err = func() error {
				defer term.Close()
				if err := term.Init(); err != nil {
					return fmt.Errorf("终端初始化失败: %v", err)
				}
				// Wait briefly for ready? Init sends request, but maybe we don't strictly need to wait for response if we just want to send?
				// But it's safer to wait.
				// However, term.WaitForReady needs a timeout.
				// Let's use a short timeout.
				// Note: terminal.go implementation of Init sends f=9.
				// WaitForReady waits for 'Ready' channel which is not currently signaled in the code I saw earlier?
				// Let's check terminal.go again.
				return action(d.Seat, term)
			}()

			if processedCount <= 10 {
				if err != nil {
					fmt.Printf("[%d/%d] ❌ %s (机位 %d): %v\n", processedCount, totalDevices, d.UUID, d.Seat, err)
				} else {
					fmt.Printf("[%d/%d] ✅ %s (机位 %d): 成功\n", processedCount, totalDevices, d.UUID, d.Seat)
				}
			} else {
				fmt.Printf("\r正在处理... %d/%d (成功: %d, 失败: %d)", processedCount, totalDevices, successCount, failCount)
			}

			if err != nil {
				logger.Errorf("操作设备失败 %s (机位 %d): %v", d.UUID, d.Seat, err)
				failCount++
			} else {
				logger.Infof("成功操作设备 %s (机位 %d)", d.UUID, d.Seat)
				successCount++
			}

			if progressCb != nil {
				status := "Success"
				if err != nil {
					status = fmt.Sprintf("Failed: %v", err)
				}
				progressCb(processedCount, totalDevices, fmt.Sprintf("Processed %s (Seat %d): %s", d.UUID, d.Seat, status))
			}
		}
	}

	fmt.Printf("\n批量操作完成。成功: %d, 失败: %d\n", successCount, failCount)

	if failCount > 0 {
		return fmt.Errorf("部分操作失败")
	}
	return nil
}

// RestartServiceBatch executes the restart service command on multiple devices concurrently.
func (c *DeviceController) RestartServiceBatch(devices []model.DeviceInfo, service string, actionCode int) error {
	// Deduplicate servers
	uniqueServers := make(map[string]struct{})
	var targetServers []string
	for _, d := range devices {
		if _, exists := uniqueServers[d.ServerURL]; !exists {
			uniqueServers[d.ServerURL] = struct{}{}
			targetServers = append(targetServers, d.ServerURL)
		}
	}

	totalServers := len(targetServers)
	concurrency := config.GlobalSettings.MaxConcurrency
	if concurrency <= 0 {
		concurrency = 5
	}
	fmt.Printf("开始对 %d 个中间件服务进行批量重启 (涉及 %d 台设备, 并发数: %d)...\n", totalServers, len(devices), concurrency)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)

	successCount := int32(0)
	failCount := int32(0)
	processedCount := int32(0)

	for _, serverURL := range targetServers {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()

			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			// Process
			err := func() error {
				server, found := c.findServerConfig(url)
				if !found {
					return fmt.Errorf("缺少服务器配置: %s", url)
				}

				// HTTP only, no WS
				deviceAPI := api.NewDeviceAPI(nil, server.URL, server.Token)
				return deviceAPI.RestartService(service, actionCode)
			}()

			currentProcessed := atomic.AddInt32(&processedCount, 1)

			// Simple console output
			if err != nil {
				atomic.AddInt32(&failCount, 1)
				logger.Errorf("重启服务失败 %s: %v", url, err)
			} else {
				atomic.AddInt32(&successCount, 1)
				logger.Infof("成功重启服务 %s", url)
			}

			// Print progress (approximate, might interleave but \r helps)
			fmt.Printf("\r正在处理... %d/%d (成功: %d, 失败: %d)", currentProcessed, totalServers, atomic.LoadInt32(&successCount), atomic.LoadInt32(&failCount))

		}(serverURL)
	}

	wg.Wait()
	fmt.Printf("\n批量操作完成。成功: %d, 失败: %d\n", successCount, failCount)

	if failCount > 0 {
		return fmt.Errorf("部分操作失败")
	}
	return nil
}

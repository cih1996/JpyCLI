package stress

import (
	"crypto/tls"
	"fmt"
	"time"

	"adminApi"
	"adminApi/changeOsCtl"
	"adminApi/loginCtl"
	"adminApi/userDeviceCtl"

	"cnb.cool/accbot/goTool/logs"
	"cnb.cool/accbot/goTool/sessionPkg"
	"cnb.cool/accbot/goTool/wsPkg"
	"github.com/gorilla/websocket"
)

// initSDKSilent 静默初始化 SDK，抑制内部日志输出到终端
func initSDKSilent(serverURL string) error {
	// 配置 SDK 日志输出到 /dev/null 或 NUL，避免干扰终端
	silentLogConfig := `
level: error
console: false
file: false
`
	_ = logs.SetLoggerConfig(silentLogConfig, 0)

	// 连接 WebSocket
	dialer := *websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	c, _, err := dialer.Dial(serverURL, nil)
	if err != nil {
		return fmt.Errorf("连接服务器失败: %w", err)
	}

	conn := wsPkg.NewWSConnByConn(c)
	sdkSession = sessionPkg.CreateSession(sessionPkg.SessionType_ws, conn)

	// 注册断线回调
	sdkSession.RegExpiryCallback(func(msg string) {
		// 静默处理，不输出到终端
	})

	sdkClient = adminApi.NewAdminApi(sdkSession)
	return nil
}

// loginWithSecret 使用密钥登录
func loginWithSecret(secretKey string) error {
	if sdkClient == nil {
		return fmt.Errorf("SDK 未初始化")
	}

	req := &loginCtl.SecretKeyLoginReq{
		SecretKey: &secretKey,
	}

	_, errPkg := sdkClient.LoginCtl.SecretKeyLogin(req)
	if errPkg != nil {
		return fmt.Errorf("登录失败: %v", errPkg)
	}
	return nil
}

// fetchAllDevices 获取所有设备
func fetchAllDevices(logger *StressLogger) ([]int64, map[int64]struct{ UUID, IP string }, error) {
	var ids []int64
	infoMap := make(map[int64]struct{ UUID, IP string })

	pageNum := 1
	pageSize := 100

	for {
		req := &userDeviceCtl.GetUserDeviceListReq{
			PageNum:  pageNum,
			PageSize: pageSize,
		}

		res, errPkg := sdkClient.UserDeviceCtl.GetUserDeviceList(req)
		if errPkg != nil {
			return nil, nil, fmt.Errorf("获取设备列表失败: %v", errPkg)
		}

		if len(res.Records) == 0 {
			break
		}

		for _, record := range res.Records {
			d := record.DeviceInfo
			ids = append(ids, d.DeviceId)
			infoMap[d.DeviceId] = struct{ UUID, IP string }{d.UUID, d.Ip}
		}

		if len(res.Records) < pageSize {
			break
		}
		pageNum++
	}

	return ids, infoMap, nil
}

// filterOnlineDevices 过滤在线设备
func filterOnlineDevices(ids []int64, logger *StressLogger) []int64 {
	idMap := make(map[int64]bool)
	for _, id := range ids {
		idMap[id] = true
	}

	onlineMap := make(map[int64]bool)
	pageNum := 1
	pageSize := 100

	for {
		req := &userDeviceCtl.GetUserDeviceListReq{
			PageNum:  pageNum,
			PageSize: pageSize,
		}

		var res *userDeviceCtl.GetUserDeviceListRes
		var errPkg interface{}

		func() {
			defer func() {
				if r := recover(); r != nil {
					errPkg = fmt.Errorf("panic: %v", r)
				}
			}()
			res, errPkg = sdkClient.UserDeviceCtl.GetUserDeviceList(req)
		}()

		if errPkg != nil {
			logger.Error("获取设备状态失败 (第 %d 页): %v", pageNum, errPkg)
			time.Sleep(5 * time.Second)
			continue
		}

		if len(res.Records) == 0 {
			break
		}

		for _, record := range res.Records {
			d := record.DeviceInfo
			if d.Online && idMap[d.DeviceId] {
				onlineMap[d.DeviceId] = true
			}
		}

		if len(res.Records) < pageSize {
			break
		}
		pageNum++
	}

	var onlineIDs []int64
	for _, id := range ids {
		if onlineMap[id] {
			onlineIDs = append(onlineIDs, id)
		} else {
			logger.Warn("设备 %d 离线，跳过", id)
		}
	}

	return onlineIDs
}

// performChangeOs 执行改机操作
func performChangeOs(deviceIDs []int64, params *ChangeOsParams, logger *StressLogger, timeout time.Duration, deviceInfoMap map[int64]struct{ UUID, IP string }) (success, failed int, results []DeviceResult) {
	logger.Info("开始为 %d 台设备发送改机请求...", len(deviceIDs))

	// 准备请求
	var reqs []*changeOsCtl.ChangeOsReq
	for _, id := range deviceIDs {
		dID := id
		req := &changeOsCtl.ChangeOsReq{
			DeviceId:     &dID,
			Bs:           &params.Bs,
			Category:     &params.Category,
			Version:      &params.Version,
			Country:      &params.Country,
			Language:     &params.Language,
			Timezone:     &params.Timezone,
			OperatorName: &params.OperatorName,
			Mcc:          &params.Mcc,
			Mnc:          &params.Mnc,
			Operator:     &params.Operator,
			Msisdn:       &params.Msisdn,
			Smsc:         &params.Smsc,
		}
		reqs = append(reqs, req)
	}

	// 分批发送请求
	var allResList []*changeOsCtl.ChangeOsRes
	batchSize := 50

	for i := 0; i < len(reqs); i += batchSize {
		end := i + batchSize
		if end > len(reqs) {
			end = len(reqs)
		}
		batchReqs := reqs[i:end]

		logger.Info("发送第 %d-%d 个请求...", i+1, end)

		maxRetries := 3
		for retry := 0; retry < maxRetries; retry++ {
			if retry > 0 {
				time.Sleep(2 * time.Second)
			}

			resList, errPkg := sdkClient.ChangeOsCtl.ChangeOs(batchReqs)
			if errPkg == nil {
				allResList = append(allResList, resList...)
				break
			}

			logger.Warn("批量请求失败 (尝试 %d/%d): %v", retry+1, maxRetries, errPkg)
			if retry == maxRetries-1 {
				logger.Error("批量改机 API 调用最终失败 (%d-%d)", i+1, end)
			}
		}

		time.Sleep(200 * time.Millisecond)
	}

	if len(allResList) == 0 {
		logger.Warn("改机请求未收到响应")
		return 0, len(deviceIDs), nil
	}

	logger.Info("收到 %d 个任务 ID，开始轮询状态...", len(allResList))

	// 构建任务映射
	taskMap := make(map[int64]int64) // changeOsID -> deviceID
	var taskIDs []int64
	for _, r := range allResList {
		changeOsID := int64(r.Id)
		taskMap[changeOsID] = r.DeviceId
		taskIDs = append(taskIDs, changeOsID)
	}

	// 轮询状态直到完成或超时
	startTime := time.Now()
	completedTasks := make(map[int64]bool)
	taskResults := make(map[int64]struct {
		status   int
		progress string
	})

	for {
		if time.Since(startTime) > timeout {
			logger.Warn("轮询超时 (%v)", timeout)
			break
		}

		// 获取未完成任务的状态
		var pendingIDs []int64
		for _, id := range taskIDs {
			if !completedTasks[id] {
				pendingIDs = append(pendingIDs, id)
			}
		}

		if len(pendingIDs) == 0 {
			break
		}

		// 分批查询状态
		for i := 0; i < len(pendingIDs); i += batchSize {
			end := i + batchSize
			if end > len(pendingIDs) {
				end = len(pendingIDs)
			}
			batchIDs := pendingIDs[i:end]

			statusReq := changeOsCtl.GetChangeOsStatusReq{
				TbChangeOsIds: &batchIDs,
			}

			var res []*changeOsCtl.GetChangeOsStatusRes
			var errPkg interface{}

			func() {
				defer func() {
					if r := recover(); r != nil {
						errPkg = fmt.Errorf("panic: %v", r)
					}
				}()
				res, errPkg = sdkClient.ChangeOsCtl.GetChangeOsStatus(statusReq)
			}()

			if errPkg != nil {
				logger.Error("获取状态失败: %v", errPkg)
				continue
			}

			for _, s := range res {
				taskID := int64(s.Id)
				status := int(s.Status)

				// status: 1=成功, -1=失败, 0=进行中
				if status == 1 || status < 0 {
					completedTasks[taskID] = true
					taskResults[taskID] = struct {
						status   int
						progress string
					}{status, s.Progress}
				}
			}
		}

		// 显示进度
		completed := len(completedTasks)
		total := len(taskIDs)
		logger.Info("进度: %d/%d (%.1f%%)", completed, total, float64(completed)*100/float64(total))

		if len(completedTasks) == len(taskIDs) {
			break
		}

		time.Sleep(3 * time.Second)
	}

	// 统计结果
	for taskID, deviceID := range taskMap {
		info := deviceInfoMap[deviceID]
		result := DeviceResult{
			DeviceID: deviceID,
			UUID:     info.UUID,
			IP:       info.IP,
		}

		if r, ok := taskResults[taskID]; ok {
			if r.status == 1 {
				result.Success = true
				success++
				logger.Info("设备 %d (%s) 改机成功", deviceID, info.UUID)
			} else {
				result.Success = false
				result.Error = r.progress
				failed++
				logger.Error("设备 %d (%s) 改机失败: %s", deviceID, info.UUID, r.progress)
			}
		} else {
			result.Success = false
			result.Error = "超时未完成"
			failed++
			logger.Error("设备 %d (%s) 改机超时", deviceID, info.UUID)
		}

		results = append(results, result)
	}

	logger.Info("本轮改机完成: 成功=%d, 失败=%d", success, failed)
	return success, failed, results
}

package stress

import (
	"crypto/tls"
	"fmt"
	"strings"
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

// isRealError 判断 SDK 返回的错误是否是真正的错误
// SDK 返回的 errPkg 类型是 interface{}，即使没有错误也可能不是 nil
func isRealError(errPkg interface{}) bool {
	if errPkg == nil {
		return false
	}
	// SDK 的错误即使为空也会返回一个非 nil 的结构体，需要转字符串判断
	errStr := fmt.Sprintf("%v", errPkg)
	return errStr != "<nil>" && errStr != ""
}

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
	if isRealError(errPkg) {
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
		if isRealError(errPkg) {
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

		if isRealError(errPkg) {
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

// checkDevicesOnline 检查指定设备是否在线
func checkDevicesOnline(deviceIDs []int64, logger *StressLogger) map[int64]bool {
	result := make(map[int64]bool)
	if len(deviceIDs) == 0 {
		return result
	}

	// 构建待查设备ID集合
	idSet := make(map[int64]bool)
	for _, id := range deviceIDs {
		idSet[id] = true
	}

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

		if isRealError(errPkg) {
			logger.Error("检查设备在线状态失败: %v", errPkg)
			break
		}

		if len(res.Records) == 0 {
			break
		}

		for _, record := range res.Records {
			d := record.DeviceInfo
			if idSet[d.DeviceId] && d.Online {
				result[d.DeviceId] = true
			}
		}

		if len(res.Records) < pageSize {
			break
		}
		pageNum++
	}

	return result
}

// performChangeOs 执行改机操作
func performChangeOs(deviceIDs []int64, params *ChangeOsParams, logger *StressLogger, timeout time.Duration, deviceInfoMap map[int64]struct{ UUID, IP string }, round int, debug bool) (success, failed int, results []DeviceResult) {
	logger.Info("[第 %d 轮] 开始为 %d 台设备发送改机请求...", round, len(deviceIDs))

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

		if len(reqs) > batchSize {
			logger.Info("发送第 %d-%d 个请求...", i+1, end)
		}

		maxRetries := 3
		for retry := 0; retry < maxRetries; retry++ {
			if retry > 0 {
				time.Sleep(2 * time.Second)
			}

			resList, errPkg := sdkClient.ChangeOsCtl.ChangeOs(batchReqs)
			if !isRealError(errPkg) {
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

	logger.Info("[第 %d 轮] 收到 %d 个任务 ID，等待 10 秒后开始轮询...", round, len(allResList))

	// 等待 10 秒，避免拉到缓存的旧状态（秒完成问题）
	time.Sleep(10 * time.Second)

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
	// 记录最后一次获取到的原始状态（用于超时时显示，包含 status 和 progress）
	lastRawStatus := make(map[int64]struct {
		status   int
		progress string
	})
	// 记录已完成改机、等待上线的设备 (deviceID -> true)
	waitingOnline := make(map[int64]bool)
	pollCount := 0

	for {
		pollCount++
		elapsed := time.Since(startTime).Round(time.Second)

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

		// 统计等待上线的设备数量
		waitingCount := 0
		for _, taskID := range pendingIDs {
			deviceID := taskMap[taskID]
			if waitingOnline[deviceID] {
				waitingCount++
			}
		}

		// 每 3 次轮询输出一次状态（减少日志量）
		if pollCount%3 == 1 {
			logger.Info("[第 %d 轮 | 轮询 #%d] 已耗时 %v, 待处理: %d 台 (其中 %d 台等待上线)", round, pollCount, elapsed, len(pendingIDs), waitingCount)
		}

		// 如果有等待上线的设备，检查它们的在线状态
		if waitingCount > 0 {
			var waitingDeviceIDs []int64
			for _, taskID := range pendingIDs {
				deviceID := taskMap[taskID]
				if waitingOnline[deviceID] {
					waitingDeviceIDs = append(waitingDeviceIDs, deviceID)
				}
			}

			// 查询这些设备的在线状态
			onlineStatus := checkDevicesOnline(waitingDeviceIDs, logger)
			for _, taskID := range pendingIDs {
				deviceID := taskMap[taskID]
				if waitingOnline[deviceID] && onlineStatus[deviceID] {
					// 设备已上线，标记为成功
					completedTasks[taskID] = true
					taskResults[taskID] = struct {
						status   int
						progress string
					}{1, "改机完成并已上线"}
					// 不再单独输出成功日志
				}
			}
		}

		// 分批查询改机任务状态
		for i := 0; i < len(pendingIDs); i += batchSize {
			end := i + batchSize
			if end > len(pendingIDs) {
				end = len(pendingIDs)
			}
			batchIDs := pendingIDs[i:end]

			// 过滤掉已经在等待上线的任务，不需要再查询改机状态
			var queryIDs []int64
			for _, id := range batchIDs {
				deviceID := taskMap[id]
				if !waitingOnline[deviceID] && !completedTasks[id] {
					queryIDs = append(queryIDs, id)
				}
			}

			if len(queryIDs) == 0 {
				continue
			}

			statusReq := changeOsCtl.GetChangeOsStatusReq{
				TbChangeOsIds: &queryIDs,
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

			if isRealError(errPkg) {
				logger.Error("获取状态失败: %v", errPkg)
				continue
			}

			for _, s := range res {
				taskID := int64(s.Id)
				deviceID := taskMap[taskID]
				status := int(s.Status)
				progress := s.Progress

				// 记录最新原始状态（无论是否完成）
				lastRawStatus[taskID] = struct {
					status   int
					progress string
				}{status, progress}

				// 状态判断：
				// status=200 + progress="改机完成" = 真正完成，进入等待上线
				// status=1 + progress="" = 可能是缓存的旧状态（秒返回），暂时认为成功但需要验证
				// status<0 = 失败
				// status=2,3等 = 进行中
				if status == 200 && strings.Contains(progress, "改机完成") {
					// 真正改机完成，进入等待上线状态（静默处理）
					if !waitingOnline[deviceID] {
						waitingOnline[deviceID] = true
					}
				} else if status < 0 {
					completedTasks[taskID] = true
					taskResults[taskID] = struct {
						status   int
						progress string
					}{status, progress}
					// 失败日志在汇总中统一输出
				}
			}
		}

		// 统计当前各状态数量
		successCount := 0
		failedCount := 0
		for _, r := range taskResults {
			if r.status == 1 {
				successCount++
			} else if r.status < 0 {
				failedCount++
			}
		}
		pendingCount := len(taskIDs) - len(completedTasks)

		// 每 10 次轮询输出详细状态汇总
		if pollCount%10 == 0 {
			logger.Info("========== 设备状态汇总 ==========")
			logger.Info("  成功: %d 台 | 失败: %d 台 | 待处理: %d 台", successCount, failedCount, pendingCount)
			if pendingCount > 0 {
				logger.Info("---------- 待处理设备 ----------")
				for taskID, deviceID := range taskMap {
					if completedTasks[taskID] {
						continue
					}
					info := deviceInfoMap[deviceID]
					raw := lastRawStatus[taskID]
					statusNote := ""
					if waitingOnline[deviceID] {
						statusNote = " [等待上线]"
					}
					logger.Info("  设备 %d | %s | status=%d | progress=%s%s", deviceID, info.UUID, raw.status, raw.progress, statusNote)
				}
			}
			if failedCount > 0 {
				logger.Info("---------- 失败设备 ----------")
				for taskID, deviceID := range taskMap {
					r, ok := taskResults[taskID]
					if !ok || r.status >= 0 {
						continue
					}
					info := deviceInfoMap[deviceID]
					logger.Info("  设备 %d | %s | status=%d | progress=%s", deviceID, info.UUID, r.status, r.progress)
				}
			}
			logger.Info("==================================")
		}

		// 每 3 次轮询显示简要进度
		if pollCount%3 == 0 && pollCount%10 != 0 {
			logger.Info("[第 %d 轮] 成功=%d 失败=%d 待处理=%d", round, successCount, failedCount, pendingCount)
		}

		if len(completedTasks) == len(taskIDs) {
			break
		}

		time.Sleep(5 * time.Second)
	}

	// 统计结果
	var failedDevices []DeviceResult
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
			} else {
				result.Success = false
				result.Error = fmt.Sprintf("status=%d, progress=%s", r.status, r.progress)
				failed++
				failedDevices = append(failedDevices, result)
			}
		} else {
			// 超时未完成，显示最后的原始状态
			raw := lastRawStatus[taskID]
			result.Success = false
			if waitingOnline[deviceID] {
				result.Error = fmt.Sprintf("超时(等待上线) | 最后状态: status=%d, progress=%s", raw.status, raw.progress)
			} else {
				result.Error = fmt.Sprintf("超时 | 最后状态: status=%d, progress=%s", raw.status, raw.progress)
			}
			failed++
			failedDevices = append(failedDevices, result)
		}

		results = append(results, result)
	}

	// 只输出失败设备汇总
	if len(failedDevices) > 0 {
		logger.Error("========== 失败设备 (%d 台) ==========", len(failedDevices))
		for _, d := range failedDevices {
			logger.Error("  设备 %d | %s | %s | %s", d.DeviceID, d.UUID, d.IP, d.Error)
		}
		logger.Error("======================================")
	}

	logger.Info("[第 %d 轮] 完成: 成功=%d, 失败=%d", round, success, failed)
	return success, failed, results
}

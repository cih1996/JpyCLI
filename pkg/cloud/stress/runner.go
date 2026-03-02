package stress

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"adminApi/changeOsCtl"
	"adminApi/userDeviceCtl"

	"jpy-cli/pkg/cloud"
)

// ChangeOsParams 改机参数（JSON 配置文件格式）
type ChangeOsParams struct {
	Bs           string `json:"bs"`
	Category     string `json:"category"`
	Version      string `json:"version"`
	Country      string `json:"country"`
	Language     string `json:"language"`
	Timezone     string `json:"timezone"`
	OperatorName string `json:"operatorName"`
	Mcc          string `json:"mcc"`
	Mnc          string `json:"mnc"`
	Operator     string `json:"operator"`
	Msisdn       string `json:"msisdn"`
	Smsc         string `json:"smsc"`
}

// DeviceInfo 设备基本信息
type DeviceInfo struct {
	DeviceID int64
	UUID     string
	Brand    string
	Version  string
	IP       string
	Online   bool
}

// TaskResult 单个改机任务结果
type TaskResult struct {
	TaskID   int64  `json:"task_id"`
	DeviceID int64  `json:"device_id"`
	Status   int    `json:"status"`   // 200=成功, <0=失败, 其他=进行中
	Progress string `json:"progress"` // 进度描述
}

// RoundSummary 单轮汇总
type RoundSummary struct {
	Round        int           `json:"round"`
	TotalDevices int           `json:"total_devices"`
	OnlineCount  int           `json:"online_count"`
	TaskCount    int           `json:"task_count"`
	SuccessCount int           `json:"success_count"`
	FailCount    int           `json:"fail_count"`
	PendingCount int           `json:"pending_count"`
	TimeoutCount int           `json:"timeout_count"`
	Duration     time.Duration `json:"duration"`
	Results      []TaskResult  `json:"results,omitempty"`
	FailReasons  []FailReason  `json:"fail_reasons,omitempty"`
}

// FailReason 失败原因统计
type FailReason struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

// StressConfig 压力测试配置
type StressConfig struct {
	ConfigPath string        // 改机参数配置文件路径
	Loop       int           // 循环次数（0=无限）
	Interval   time.Duration // 轮次间隔
	Timeout    time.Duration // 单轮超时
	BatchSize  int           // 批次大小
	MaxRetries int           // 最大重试次数
	// 通知配置
	LarkWebhook             string // 飞书 Webhook URL
	NotifyEnabled           bool   // 是否启用通知
	NotifyAlways            bool   // 每轮都通知（否则仅掉线率超阈值时通知）
	OfflineThresholdPercent int    // 掉线率阈值百分比
	// 日志配置
	LogDir string // 日志目录，空则不写文件
}

// ProgressCallback 进度回调
type ProgressCallback func(event string, data interface{})

// Runner 压力测试执行器
type Runner struct {
	config   StressConfig
	devices  []DeviceInfo
	callback ProgressCallback
	logger   func(level, format string, args ...interface{})
	logFile  *os.File // 运行日志文件
	taskID   string   // 任务编号，如 TASK-20060102150405
}

// NewRunner 创建压力测试执行器
func NewRunner(cfg StressConfig, devices []DeviceInfo, callback ProgressCallback) *Runner {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}
	r := &Runner{
		config:   cfg,
		devices:  devices,
		callback: callback,
		taskID:   fmt.Sprintf("TASK-%s", time.Now().Format("20060102150405")),
		logger: func(level, format string, args ...interface{}) {
			// 默认日志：静默
		},
	}

	// 初始化日志文件
	if cfg.LogDir != "" {
		if err := os.MkdirAll(cfg.LogDir, 0755); err == nil {
			logPath := fmt.Sprintf("%s/change_os_%s.log", cfg.LogDir, time.Now().Format("20060102_150405"))
			if f, err := os.Create(logPath); err == nil {
				r.logFile = f
			}
		}
	}

	return r
}

// Close 关闭资源
func (r *Runner) Close() {
	if r.logFile != nil {
		r.logFile.Close()
	}
}

// SetLogger 设置日志函数
func (r *Runner) SetLogger(logger func(level, format string, args ...interface{})) {
	r.logger = logger
}

// GetAllDevices 获取所有设备列表（使用用户端接口，普通用户可用）
func GetAllDevices() ([]DeviceInfo, error) {
	if err := cloud.EnsureConnected(); err != nil {
		return nil, err
	}

	var devices []DeviceInfo
	pageNum := 1
	pageSize := 100

	for {
		req := &userDeviceCtl.GetUserDeviceListReq{
			PageNum:  pageNum,
			PageSize: pageSize,
		}
		res, errPkg := cloud.Client.UserDeviceCtl.GetUserDeviceList(req)
		if errPkg != nil {
			return nil, fmt.Errorf("获取设备列表失败 (第 %d 页): %v", pageNum, errPkg)
		}

		if len(res.Records) == 0 {
			break
		}

		for _, d := range res.Records {
			di := d.DeviceInfo
			devices = append(devices, DeviceInfo{
				DeviceID: di.DeviceId,
				UUID:     di.UUID,
				Brand:    di.Brand,
				Version:  di.Version,
				IP:       di.Ip,
				Online:   di.Online,
			})
		}

		if len(res.Records) < pageSize {
			break
		}
		pageNum++
	}

	return devices, nil
}

// Run 执行压力测试
func (r *Runner) Run() ([]RoundSummary, error) {
	defer r.Close()

	// 读取改机配置
	params, err := r.loadChangeOsParams()
	if err != nil {
		return nil, err
	}

	deviceIDs := make([]int64, len(r.devices))
	for i, d := range r.devices {
		deviceIDs[i] = d.DeviceID
	}

	r.writeLog("INFO", "任务 %s 启动，设备数: %d, 配置: %s", r.taskID, len(deviceIDs), r.config.ConfigPath)

	var summaries []RoundSummary
	round := 1

	for {
		r.emit("round_start", map[string]interface{}{"round": round, "total": len(deviceIDs)})
		r.logger("INFO", "=== 开始第 %d 轮 ===", round)
		r.writeLog("INFO", "=== 开始第 %d 轮 ===", round)

		startTime := time.Now()

		// 1. 过滤在线设备
		onlineIDs := r.filterOnlineDevices(deviceIDs)
		r.emit("online_check_done", map[string]interface{}{
			"round": round, "total": len(deviceIDs), "online": len(onlineIDs),
		})
		r.logger("INFO", "选中 %d 台设备，其中 %d 台在线", len(deviceIDs), len(onlineIDs))
		r.writeLog("INFO", "选中 %d 台设备，其中 %d 台在线", len(deviceIDs), len(onlineIDs))

		summary := RoundSummary{
			Round:        round,
			TotalDevices: len(deviceIDs),
			OnlineCount:  len(onlineIDs),
		}

		if len(onlineIDs) > 0 {
			// 2. 发送改机请求
			results := r.performChangeOs(onlineIDs, params)
			summary.TaskCount = len(results)

			// 3. 轮询状态直到完成或超时
			finalResults := r.pollStatus(results, round)

			// 4. 统计
			for _, tr := range finalResults {
				switch {
				case tr.Status == 200:
					summary.SuccessCount++
				case tr.Status < 0:
					summary.FailCount++
					r.writeLog("ERROR", "设备 %d 改机失败: %s", tr.DeviceID, tr.Progress)
				default:
					summary.TimeoutCount++ // 超时仍未完成
				}
			}
			summary.PendingCount = summary.TimeoutCount
			summary.Results = finalResults

			// 统计失败原因
			summary.FailReasons = r.aggregateFailReasons(finalResults)
		}

		summary.Duration = time.Since(startTime)
		summaries = append(summaries, summary)

		r.emit("round_done", summary)
		r.logger("INFO", "第 %d 轮完成 - 总计: %d, 成功: %d, 失败: %d, 超时: %d, 耗时: %v",
			round, summary.TaskCount, summary.SuccessCount, summary.FailCount, summary.TimeoutCount, summary.Duration)
		r.writeLog("INFO", "第 %d 轮完成 - 总计: %d, 成功: %d, 失败: %d, 超时: %d, 耗时: %v",
			round, summary.TaskCount, summary.SuccessCount, summary.FailCount, summary.TimeoutCount, summary.Duration)

		// 发送飞书通知
		r.sendLarkNotification(summary)

		// 检查是否继续
		if r.config.Loop > 0 && round >= r.config.Loop {
			break
		}

		// 等待间隔
		r.emit("interval_wait", map[string]interface{}{
			"round": round, "interval": r.config.Interval.String(),
		})
		r.logger("INFO", "第 %d 轮结束，等待 %v...", round, r.config.Interval)
		r.writeLog("INFO", "第 %d 轮结束，等待 %v...", round, r.config.Interval)
		time.Sleep(r.config.Interval)
		round++
	}

	return summaries, nil
}

func (r *Runner) loadChangeOsParams() (*ChangeOsParams, error) {
	data, err := os.ReadFile(r.config.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("读取改机配置文件失败: %w", err)
	}

	var params ChangeOsParams
	if err := json.Unmarshal(data, &params); err != nil {
		return nil, fmt.Errorf("解析改机配置文件失败: %w", err)
	}
	return &params, nil
}

// filterOnlineDevices 过滤在线设备（用户端接口不支持 Online 过滤参数，本地遍历过滤）
func (r *Runner) filterOnlineDevices(ids []int64) []int64 {
	idMap := make(map[int64]bool)
	for _, id := range ids {
		idMap[id] = true
	}

	foundOnline := make(map[int64]bool)
	pageNum := 1
	pageSize := 100

	for {
		req := &userDeviceCtl.GetUserDeviceListReq{
			PageNum:  pageNum,
			PageSize: pageSize,
		}

		res, errPkg := cloud.Client.UserDeviceCtl.GetUserDeviceList(req)
		if errPkg != nil {
			r.logger("ERROR", "获取设备列表失败 (第 %d 页): %v", pageNum, errPkg)
			break
		}

		if len(res.Records) == 0 {
			break
		}

		for _, d := range res.Records {
			di := d.DeviceInfo
			if di.Online && idMap[di.DeviceId] {
				foundOnline[di.DeviceId] = true
			}
		}

		if len(res.Records) < pageSize {
			break
		}
		pageNum++
	}

	var online []int64
	for _, id := range ids {
		if foundOnline[id] {
			online = append(online, id)
		} else {
			r.logger("WARN", "设备 %d 离线，跳过", id)
		}
	}
	return online
}

func (r *Runner) performChangeOs(deviceIDs []int64, params *ChangeOsParams) []TaskResult {
	// 准备请求（使用用户端 changeOsCtl 接口）
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

	// 分批发送
	var allResults []TaskResult
	batchSize := r.config.BatchSize

	for i := 0; i < len(reqs); i += batchSize {
		end := i + batchSize
		if end > len(reqs) {
			end = len(reqs)
		}
		batchReqs := reqs[i:end]

		r.emit("batch_send", map[string]interface{}{
			"from": i + 1, "to": end, "total": len(reqs),
		})
		r.logger("INFO", "正在发送第 %d-%d 个请求...", i+1, end)

		for retry := 0; retry < r.config.MaxRetries; retry++ {
			if retry > 0 {
				time.Sleep(2 * time.Second)
			}

			resList, errPkg := cloud.Client.ChangeOsCtl.ChangeOs(batchReqs)
			if errPkg == nil {
				for _, res := range resList {
					allResults = append(allResults, TaskResult{
						TaskID:   int64(res.Id),
						DeviceID: res.DeviceId,
						Status:   0, // 初始状态
						Progress: "已提交",
					})
				}
				break
			}

			r.logger("WARN", "批量请求失败 (尝试 %d/%d): %v", retry+1, r.config.MaxRetries, errPkg)
			if retry == r.config.MaxRetries-1 {
				r.logger("ERROR", "批量改机请求最终失败 (%d-%d)", i+1, end)
			}
		}

		time.Sleep(200 * time.Millisecond) // 批次间隔
	}

	r.emit("all_sent", map[string]interface{}{"task_count": len(allResults)})
	r.logger("INFO", "请求发送完毕，共 %d 个任务", len(allResults))

	return allResults
}

func (r *Runner) pollStatus(tasks []TaskResult, round int) []TaskResult {
	if len(tasks) == 0 {
		return tasks
	}

	taskIDs := make([]int64, len(tasks))
	taskMap := make(map[int64]*TaskResult)
	for i := range tasks {
		taskIDs[i] = tasks[i].TaskID
		taskMap[tasks[i].TaskID] = &tasks[i]
	}

	deadline := time.Now().Add(r.config.Timeout)
	pollInterval := 2 * time.Second

	for {
		if time.Now().After(deadline) {
			r.logger("WARN", "轮次 %d 超时，停止轮询", round)
			// 标记未完成的任务为超时
			for i := range tasks {
				if tasks[i].Status >= 0 && tasks[i].Status != 200 {
					tasks[i].Status = -999
					tasks[i].Progress = "超时"
				}
			}
			break
		}

		// 分批查询状态
		allDone := true
		var mu sync.Mutex

		for i := 0; i < len(taskIDs); i += r.config.BatchSize {
			end := i + r.config.BatchSize
			if end > len(taskIDs) {
				end = len(taskIDs)
			}
			batchIDs := taskIDs[i:end]

			statusReq := changeOsCtl.GetChangeOsStatusReq{
				TbChangeOsIds: &batchIDs,
			}

			res, errPkg := cloud.Client.ChangeOsCtl.GetChangeOsStatus(statusReq)
			if errPkg != nil {
				r.logger("ERROR", "获取状态失败: %v", errPkg)
				continue
			}

			mu.Lock()
			for _, s := range res {
				if t, ok := taskMap[int64(s.Id)]; ok {
					t.Status = int(s.Status)
					t.Progress = s.Progress
					if s.Status >= 0 && s.Status != 200 {
						allDone = false
					}
				}
			}
			mu.Unlock()
		}

		// 计算当前统计
		success, fail, pending := 0, 0, 0
		for _, t := range tasks {
			switch {
			case t.Status == 200:
				success++
			case t.Status < 0:
				fail++
			default:
				pending++
			}
		}

		r.emit("poll_update", map[string]interface{}{
			"round": round, "success": success, "fail": fail, "pending": pending, "total": len(tasks),
		})

		if allDone || pending == 0 {
			break
		}

		time.Sleep(pollInterval)
	}

	return tasks
}

func (r *Runner) aggregateFailReasons(results []TaskResult) []FailReason {
	reasonCount := make(map[string]int)
	for _, tr := range results {
		if tr.Status < 0 {
			reason := tr.Progress
			if reason == "" {
				reason = "未知错误"
			}
			// 截断过长原因
			runes := []rune(reason)
			if len(runes) > 50 {
				reason = string(runes[:50]) + "..."
			}
			reasonCount[reason]++
		}
	}

	var reasons []FailReason
	for reason, count := range reasonCount {
		reasons = append(reasons, FailReason{Reason: reason, Count: count})
	}
	sort.Slice(reasons, func(i, j int) bool {
		return reasons[i].Count > reasons[j].Count
	})

	return reasons
}

func (r *Runner) emit(event string, data interface{}) {
	if r.callback != nil {
		r.callback(event, data)
	}
}

// writeLog 写入日志文件
func (r *Runner) writeLog(level, format string, args ...interface{}) {
	if r.logFile != nil {
		msg := fmt.Sprintf(format, args...)
		line := fmt.Sprintf("[%s] [%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), level, msg)
		r.logFile.WriteString(line)
	}
}

// sendLarkNotification 发送飞书通知
func (r *Runner) sendLarkNotification(summary RoundSummary) {
	if !r.config.NotifyEnabled || r.config.LarkWebhook == "" {
		return
	}

	// 判断是否需要通知
	offlineRate := 0
	if summary.TotalDevices > 0 {
		offlineRate = (summary.TotalDevices - summary.OnlineCount) * 100 / summary.TotalDevices
	}

	if !r.config.NotifyAlways && offlineRate < r.config.OfflineThresholdPercent {
		return
	}

	// 构建通知内容
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("任务编号: %s\n", r.taskID))
	sb.WriteString(fmt.Sprintf("轮次: %d\n", summary.Round))

	if r.config.NotifyAlways {
		sb.WriteString("触发: 每轮通知\n")
	} else {
		sb.WriteString(fmt.Sprintf("触发: 掉线率 %d%% (阈值 %d%%)\n", offlineRate, r.config.OfflineThresholdPercent))
	}

	sb.WriteString(fmt.Sprintf("设备: 总 %d / 在线 %d / 离线率 %d%%\n",
		summary.TotalDevices, summary.OnlineCount, offlineRate))
	sb.WriteString(fmt.Sprintf("本轮: 成功 %d / 失败 %d / 超时 %d\n",
		summary.SuccessCount, summary.FailCount, summary.TimeoutCount))

	if len(summary.FailReasons) > 0 {
		sb.WriteString("失败原因 TOP5:\n")
		limit := 5
		if len(summary.FailReasons) < limit {
			limit = len(summary.FailReasons)
		}
		for i := 0; i < limit; i++ {
			sb.WriteString(fmt.Sprintf("  %d. %s (%d次)\n", i+1, summary.FailReasons[i].Reason, summary.FailReasons[i].Count))
		}
	}

	go sendLarkWebhook(r.config.LarkWebhook, sb.String())
}

// sendLarkWebhook 异步发送飞书 Webhook
func sendLarkWebhook(webhook, content string) {
	payload := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": content,
		},
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	resp, err := http.Post(webhook, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

// FormatSummaryPlain 格式化汇总为纯文本
func FormatSummaryPlain(summaries []RoundSummary) string {
	var sb strings.Builder

	for _, s := range summaries {
		sb.WriteString(fmt.Sprintf("=== 第 %d 轮 ===\n", s.Round))
		sb.WriteString(fmt.Sprintf("设备总数: %d | 在线: %d\n", s.TotalDevices, s.OnlineCount))
		sb.WriteString(fmt.Sprintf("任务总数: %d | 成功: %d | 失败: %d | 超时: %d\n",
			s.TaskCount, s.SuccessCount, s.FailCount, s.TimeoutCount))
		sb.WriteString(fmt.Sprintf("耗时: %v\n", s.Duration.Round(time.Second)))

		if len(s.FailReasons) > 0 {
			sb.WriteString("失败原因 TOP 10:\n")
			limit := 10
			if len(s.FailReasons) < limit {
				limit = len(s.FailReasons)
			}
			for i := 0; i < limit; i++ {
				sb.WriteString(fmt.Sprintf("  %d. %s: %d\n", i+1, s.FailReasons[i].Reason, s.FailReasons[i].Count))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatSummaryJSON 格式化汇总为 JSON
func FormatSummaryJSON(summaries []RoundSummary) (string, error) {
	data, err := json.MarshalIndent(summaries, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

package selector

import (
	"context"
	"fmt"
	"jpy-cli/pkg/logger"
	"jpy-cli/pkg/middleware/connector"
	"jpy-cli/pkg/middleware/device/fetcher"
	"jpy-cli/pkg/middleware/model"
	"strings"
)

type SelectionOptions struct {
	Servers []connector.ServerInfo // 直传服务器凭证

	// 设备过滤条件
	UUID string
	IP   string
	Seat int // -1 for any
	ADB  *bool
	USB  *bool

	Context context.Context
}

// SelectDevices 获取并过滤设备列表（无状态，无 TUI）
func SelectDevices(opts SelectionOptions) ([]model.DeviceInfo, error) {
	if len(opts.Servers) == 0 {
		return nil, fmt.Errorf("未指定服务器")
	}

	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}

	resultsChan, _ := fetcher.FetchDevices(ctx, opts.Servers)

	// 静默收集结果
	var rawResults []interface{}
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case res, ok := <-resultsChan:
			if !ok {
				goto Done
			}
			rawResults = append(rawResults, res)
		}
	}
Done:

	allDevices, _ := fetcher.ProcessResults(rawResults)

	// 过滤
	var filtered []model.DeviceInfo
	for _, d := range allDevices {
		if opts.UUID != "" && !strings.Contains(d.UUID, opts.UUID) {
			continue
		}
		if opts.IP != "" && !strings.Contains(d.IP, opts.IP) {
			continue
		}
		if opts.Seat > -1 && d.Seat != opts.Seat {
			continue
		}
		if opts.ADB != nil && d.ADBEnabled != *opts.ADB {
			continue
		}
		if opts.USB != nil && d.USBMode != *opts.USB {
			continue
		}
		filtered = append(filtered, d)
	}

	logger.Infof("Selector: input %d, output %d", len(allDevices), len(filtered))
	return filtered, nil
}

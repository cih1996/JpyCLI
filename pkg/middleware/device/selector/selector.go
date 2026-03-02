package selector

import (
	"context"
	"fmt"
	"strings"
	"sync"

	httpclient "jpy-cli/pkg/client/http"
	"jpy-cli/pkg/config"
	"jpy-cli/pkg/logger"
	"jpy-cli/pkg/middleware/device/fetcher"
	"jpy-cli/pkg/middleware/model"
	"jpy-cli/pkg/tui"

	tea "github.com/charmbracelet/bubbletea"
)

type SelectionOptions struct {
	Group          string
	ServerPattern  string
	UUID           string
	Seat           int // -1 for any
	ADB            *bool
	USB            *bool
	BizOnline      *bool
	HasIP          *bool
	HasUUID        *bool
	AuthorizedOnly *bool
	Interactive    bool
	Silent         bool

	// Context for cancellation
	Context context.Context

	// Optional callback for progress updates (found, total, message)
	ProgressCallback func(int, int, string)
}

// MatchServerPattern checks if the server URL matches the pattern.
// Supports multiple patterns separated by "|".
func MatchServerPattern(url, pattern string) bool {
	if pattern == "" {
		return true
	}
	// Support OR logic with "|"
	parts := strings.Split(pattern, "|")
	for _, part := range parts {
		if part == "" {
			continue
		}
		if strings.Contains(url, part) {
			return true
		}
	}
	return false
}

// SelectDevices runs the discovery and filtering process.
// It returns a list of devices matching the criteria.
func SelectDevices(opts SelectionOptions) ([]model.DeviceInfo, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %v", err)
	}

	targetGroup := opts.Group
	if targetGroup == "" {
		targetGroup = cfg.ActiveGroup
	}
	if targetGroup == "" {
		targetGroup = "default"
	}

	allServers := config.GetGroupServers(cfg, targetGroup)
	if len(allServers) == 0 {
		return nil, fmt.Errorf("分组 '%s' 中未找到服务器", targetGroup)
	}

	// 1. Filter Servers
	var targetServers []config.LocalServerConfig
	for _, s := range allServers {
		// Skip soft-deleted servers
		if s.Disabled {
			continue
		}
		if MatchServerPattern(s.URL, opts.ServerPattern) {
			targetServers = append(targetServers, s)
		}
	}

	if len(targetServers) == 0 {
		return nil, fmt.Errorf("未找到匹配的服务器")
	}

	// 1.1 Filter Servers by Authorization (if requested)
	if opts.AuthorizedOnly != nil {
		targetServers = filterServersByAuth(targetServers, opts.AuthorizedOnly)
		if len(targetServers) == 0 {
			return nil, fmt.Errorf("没有符合授权条件的服务器")
		}
	}

	// 2. Fetch Devices (with Progress TUI or Silent)
	// We use the shared fetcher which returns a channel
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}
	resultsChan, total := fetcher.FetchDevices(ctx, targetServers, cfg)

	var rawResults []interface{}

	if opts.Silent {
		// Silent mode: just collect results
		processed := 0
		totalDevicesFound := 0
		for {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case res, ok := <-resultsChan:
				if !ok {
					goto Done
				}
				processed++
				serverRes := res.(fetcher.ServerResult)
				totalDevicesFound += len(serverRes.Devices)

				if opts.ProgressCallback != nil {
					cleanURL := strings.TrimPrefix(serverRes.ServerURL, "https://")
					cleanURL = strings.TrimPrefix(cleanURL, "http://")
					msg := fmt.Sprintf("Checked %s: Found %d devices", cleanURL, len(serverRes.Devices))
					if serverRes.Error != nil {
						msg = fmt.Sprintf("Error %s: %v", cleanURL, serverRes.Error)
					}
					opts.ProgressCallback(processed, total, msg)
				}

				rawResults = append(rawResults, res)
			}
		}
	Done:
	} else {
		// Reuse existing progress TUI
		totalDevicesFound := 0
		prog := tea.NewProgram(tui.NewProgressModel(total, resultsChan, func(v interface{}) string {
			res := v.(fetcher.ServerResult)
			cleanURL := strings.TrimPrefix(res.ServerURL, "https://")
			cleanURL = strings.TrimPrefix(cleanURL, "http://")
			if res.Error != nil {
				return fmt.Sprintf("❌ %s: %v", cleanURL, res.Error)
			}
			totalDevicesFound += len(res.Devices)
			return fmt.Sprintf("✅ %s: 发现 %d 台设备 (总计: %d)", cleanURL, len(res.Devices), totalDevicesFound)
		}))

		finalModel, err := prog.Run()
		if err != nil {
			return nil, fmt.Errorf("TUI error: %v", err)
		}
		rawResults = finalModel.(tui.ProgressModel).GetResults()
	}

	// 3. Process Results
	allDevices, _ := fetcher.ProcessResults(rawResults)

	// 4. Filter Devices
	if opts.USB != nil {
		logger.Infof("DEBUG: Applying USB Filter: required=%v", *opts.USB)
	}

	var filtered []model.DeviceInfo
	for _, d := range allDevices {
		// UUID Filter (Fuzzy)
		if opts.UUID != "" && !strings.Contains(d.UUID, opts.UUID) {
			continue
		}
		// Seat Filter
		if opts.Seat > -1 && d.Seat != opts.Seat {
			continue
		}
		// Status Filters
		if opts.ADB != nil && d.ADBEnabled != *opts.ADB {
			continue
		}
		if opts.USB != nil {
			if d.USBMode != *opts.USB {
				continue
			}
		}
		if opts.BizOnline != nil && d.BizOnline != *opts.BizOnline {
			continue
		}
		// IP Filter
		if opts.HasIP != nil {
			hasIP := d.IP != ""
			if hasIP != *opts.HasIP {
				continue
			}
		}
		// UUID Filter
		if opts.HasUUID != nil {
			hasUUID := d.UUID != ""
			if hasUUID != *opts.HasUUID {
				continue
			}
		}
		filtered = append(filtered, d)
	}

	logger.Infof("DEBUG: Selector finished. Input: %d, Output: %d", len(allDevices), len(filtered))

	if len(filtered) == 0 {
		return nil, fmt.Errorf("没有匹配的设备")
	}

	if opts.Interactive {
		return RunInteractiveSelection(filtered)
	}

	return filtered, nil
}

func filterServersByAuth(servers []config.LocalServerConfig, authorizedOnly *bool) []config.LocalServerConfig {
	if authorizedOnly == nil {
		return servers
	}
	reqAuth := *authorizedOnly

	var wg sync.WaitGroup
	var mu sync.Mutex
	var result []config.LocalServerConfig
	sem := make(chan struct{}, 20) // Limit concurrency

	for _, s := range servers {
		wg.Add(1)
		go func(server config.LocalServerConfig) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Create client without token (status check shouldn't require auth token for the CLI itself, usually)
			// Or if it does, we assume it works or we need a way to get it.
			// Based on `auto-auth`, it uses empty token.
			client := httpclient.NewClient(server.URL, "")
			info, err := client.GetLicense()
			isAuthorized := (err == nil && info != nil && info.StatusTxt == "成功")

			if isAuthorized == reqAuth {
				mu.Lock()
				result = append(result, server)
				mu.Unlock()
			}
		}(s)
	}
	wg.Wait()
	return result
}

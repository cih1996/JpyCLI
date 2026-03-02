package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"jpy-cli/pkg/config"
	"jpy-cli/pkg/middleware/device/controller"
	"jpy-cli/pkg/middleware/device/selector"
	"jpy-cli/pkg/middleware/device/status"
	"strings"
)

type DeviceListStats struct {
	Total     int `json:"total"`
	Online    int `json:"online"`
	BizOnline int `json:"biz_online"`
	USB       int `json:"usb"`
	OTG       int `json:"otg"`
	ADB       int `json:"adb"`
}

type DeviceListResponse struct {
	Items any             `json:"items"`
	Stats DeviceListStats `json:"stats"`
}

type ServerStatusStats struct {
	Total     int `json:"total"`
	Online    int `json:"online"`
	Offline   int `json:"offline"`
	Disabled  int `json:"disabled"`
	Devices   int `json:"devices"`
	BizOnline int `json:"bizOnline"`
	IPCount   int `json:"ipCount"`
	UUIDCount int `json:"uuidCount"`
	ADBCount  int `json:"adbCount"`
	USBCount  int `json:"usbCount"`
	OTGCount  int `json:"otgCount"`
}

type ServerStatusResponse struct {
	Items any               `json:"items"`
	Stats ServerStatusStats `json:"stats"`
}

type ProgressData struct {
	Current int    `json:"current"`
	Total   int    `json:"total"`
	Message string `json:"message"`
	Percent int    `json:"percent"`
}

func sendProgress(conn *SafeConn, id string, current, total int, msg string) {
	percent := 0
	if total > 0 {
		percent = int(float64(current) / float64(total) * 100)
	}
	resp := Response{
		ID:     id,
		Status: "progress",
		Data: ProgressData{
			Current: current,
			Total:   total,
			Message: msg,
			Percent: percent,
		},
	}
	conn.WriteJSON(resp)
}

func init() {
	RegisterHandler("middleware.device.list", handleDeviceList)
	RegisterHandler("middleware.device.status", handleDeviceStatus)
	RegisterHandler("middleware.device.reboot", handleDeviceReboot)
	RegisterHandler("middleware.device.usb", handleDeviceUSB)
	RegisterHandler("middleware.device.adb", handleDeviceADB)
	RegisterHandler("config.list", handleConfigList)
	RegisterHandler("config.get", handleConfigGet)
	RegisterHandler("config.set", handleConfigSet)
	RegisterHandler("config.servers.list", handleServerList)
	RegisterHandler("config.server.save", handleServerSave)
	RegisterHandler("config.server.remove", handleServerRemove)
	RegisterHandler("config.server.remove_batch", handleServerRemoveBatch)
	RegisterHandler("config.server.relogin", handleServerRelogin)
	RegisterHandler("config.server.auto_auth", handleServerAutoAuth)
	RegisterHandler("config.server.update_cluster", handleServerUpdateCluster)
}

type DeviceListParams struct {
	Group         string `json:"group"`
	ServerPattern string `json:"server"`
	UUID          string `json:"uuid"`
	Seat          *int   `json:"seat"` // Pointer to distinguish between 0 and missing

	FilterADB    string `json:"filter_adb"`
	FilterUSB    string `json:"filter_usb"`
	FilterOnline string `json:"filter_online"`
	FilterHasIP  string `json:"filter_has_ip"`
	FilterUUID   string `json:"filter_has_uuid"`

	AuthorizedOnly string `json:"authorized_only"`

	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

type DeviceStatusParams struct {
	DeviceListParams

	// Additional Status Filters
	Detail             bool     `json:"detail"`
	ShowDisabled       bool     `json:"show_disabled"`
	DisabledOnly       bool     `json:"disabled_only"`
	AuthFailed         bool     `json:"auth_failed"`
	ClusterContains    string   `json:"cluster_contains"`
	ClusterNotContains string   `json:"cluster_not_contains"`
	FwVersionHas       string   `json:"fw_version_has"`
	FwVersionNot       string   `json:"fw_version_not"`
	NetSpeedGT         *float64 `json:"net_speed_gt"`
	NetSpeedLT         *float64 `json:"net_speed_lt"`

	BizOnlineGT *int   `json:"biz_online_gt"`
	BizOnlineLT *int   `json:"biz_online_lt"`
	IPCountGT   *int   `json:"ip_count_gt"`
	IPCountLT   *int   `json:"ip_count_lt"`
	UUIDCountGT *int   `json:"uuid_count_gt"`
	UUIDCountLT *int   `json:"uuid_count_lt"`
	SNGT        string `json:"sn_gt"`
	SNLT        string `json:"sn_lt"`
}

type DeviceControlParams struct {
	DeviceListParams
	Mode  string `json:"mode"`  // For USB: host/device
	State string `json:"state"` // For ADB: on/off
}

func (p *DeviceListParams) ToSelectorOptions() selector.SelectionOptions {
	seat := -1
	if p.Seat != nil {
		seat = *p.Seat
	}

	var authPtr *bool
	switch p.AuthorizedOnly {
	case "true":
		val := true
		authPtr = &val
	case "false":
		val := false
		authPtr = &val
	}

	opts := selector.SelectionOptions{
		Group:          p.Group,
		ServerPattern:  p.ServerPattern,
		UUID:           p.UUID,
		Seat:           seat,
		AuthorizedOnly: authPtr,
		Silent:         true,
		Interactive:    false,
	}

	if p.FilterADB != "" {
		val := p.FilterADB == "true"
		opts.ADB = &val
	}
	if p.FilterUSB != "" {
		val := p.FilterUSB == "true"
		opts.USB = &val
	}
	if p.FilterOnline != "" {
		val := p.FilterOnline == "true"
		opts.BizOnline = &val
	}
	if p.FilterHasIP != "" {
		val := p.FilterHasIP == "true"
		opts.HasIP = &val
	}
	if p.FilterUUID != "" {
		val := p.FilterUUID == "true"
		opts.HasUUID = &val
	}

	return opts
}

func (p *DeviceStatusParams) ToStatusFilters() status.StatusFilters {
	seat := -1
	if p.Seat != nil {
		seat = *p.Seat
	}

	var authPtr *bool
	switch p.AuthorizedOnly {
	case "true":
		val := true
		authPtr = &val
	case "false":
		val := false
		authPtr = &val
	}

	var onlinePtr *bool
	if p.FilterOnline != "" {
		val := p.FilterOnline == "true"
		onlinePtr = &val
	}

	var adbPtr *bool
	if p.FilterADB != "" {
		val := p.FilterADB == "true"
		adbPtr = &val
	}

	var usbPtr *bool
	if p.FilterUSB != "" {
		val := p.FilterUSB == "true"
		usbPtr = &val
	}

	var hasIPPtr *bool
	if p.FilterHasIP != "" {
		val := p.FilterHasIP == "true"
		hasIPPtr = &val
	}

	var netSpeedGT, netSpeedLT float64 = -1, -1
	if p.NetSpeedGT != nil {
		netSpeedGT = *p.NetSpeedGT
	}
	if p.NetSpeedLT != nil {
		netSpeedLT = *p.NetSpeedLT
	}

	var bizOnlineGT, bizOnlineLT int = -1, -1
	if p.BizOnlineGT != nil {
		bizOnlineGT = *p.BizOnlineGT
	}
	if p.BizOnlineLT != nil {
		bizOnlineLT = *p.BizOnlineLT
	}

	var ipCountGT, ipCountLT int = -1, -1
	if p.IPCountGT != nil {
		ipCountGT = *p.IPCountGT
	}
	if p.IPCountLT != nil {
		ipCountLT = *p.IPCountLT
	}

	var uuidCountGT, uuidCountLT int = -1, -1
	if p.UUIDCountGT != nil {
		uuidCountGT = *p.UUIDCountGT
	}
	if p.UUIDCountLT != nil {
		uuidCountLT = *p.UUIDCountLT
	}

	return status.StatusFilters{
		Group:              p.Group,
		ServerPattern:      p.ServerPattern,
		AuthorizedOnly:     authPtr,
		ShowDisabled:       p.ShowDisabled,
		DisabledOnly:       p.DisabledOnly,
		AuthFailed:         p.AuthFailed,
		ClusterContains:    p.ClusterContains,
		ClusterNotContains: p.ClusterNotContains,
		FwVersionHas:       p.FwVersionHas,
		FwVersionNot:       p.FwVersionNot,
		NetSpeedGT:         netSpeedGT,
		NetSpeedLT:         netSpeedLT,
		BizOnlineGT:        bizOnlineGT,
		BizOnlineLT:        bizOnlineLT,
		IPCountGT:          ipCountGT,
		IPCountLT:          ipCountLT,
		UUIDCountGT:        uuidCountGT,
		UUIDCountLT:        uuidCountLT,
		UUID:               p.UUID,
		Seat:               seat,
		FilterOnline:       onlinePtr,
		FilterADB:          adbPtr,
		FilterUSB:          usbPtr,
		FilterHasIP:        hasIPPtr,
		Detail:             p.Detail,
		SNGT:               p.SNGT,
		SNLT:               p.SNLT,
	}
}

func handleDeviceList(conn *SafeConn, req Request) {
	var params DeviceListParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Invalid params: %v", err))
		return
	}

	opts := params.ToSelectorOptions()
	opts.ProgressCallback = func(curr, total int, msg string) {
		sendProgress(conn, req.ID, curr, total, msg)
	}

	devices, err := selector.SelectDevices(opts)
	if err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Error fetching devices: %v", err))
		return
	}

	// Calculate stats
	stats := DeviceListStats{
		Total: len(devices),
	}
	for _, d := range devices {
		if d.IsOnline {
			stats.Online++
		}
		if d.BizOnline {
			stats.BizOnline++
		}
		if d.USBMode {
			stats.USB++
		} else {
			stats.OTG++
		}
		if d.ADBEnabled {
			stats.ADB++
		}
	}

	// Pagination
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 50 // Default to 50 items per page
	}

	start := (page - 1) * pageSize
	end := start + pageSize

	// Use slice of original type
	pagedDevices := devices[:0]
	if start < len(devices) {
		if end > len(devices) {
			end = len(devices)
		}
		pagedDevices = devices[start:end]
	}

	sendSuccess(conn, req.ID, DeviceListResponse{
		Items: pagedDevices,
		Stats: stats,
	})
}

func handleDeviceStatus(conn *SafeConn, req Request) {
	var params DeviceStatusParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Invalid params: %v", err))
		return
	}

	cfg, err := config.Load()
	if err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Failed to load config: %v", err))
		return
	}

	filters := params.ToStatusFilters()
	ctx := context.Background()

	results, err := status.GetServerStatusStats(ctx, cfg, filters, func(curr, total int, msg string) {
		sendProgress(conn, req.ID, curr, total, msg)
	})
	if err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Error fetching status: %v", err))
		return
	}

	// Calculate stats
	sStats := ServerStatusStats{}
	for _, s := range results {
		if s.Disabled {
			sStats.Disabled++
		} else {
			sStats.Total++
			if s.Status == "Online" {
				sStats.Online++
			} else {
				sStats.Offline++
			}
			sStats.Devices += s.DeviceCount
			sStats.BizOnline += s.BizOnlineCount
			sStats.IPCount += s.IPCount
			sStats.UUIDCount += s.UUIDCount
			sStats.ADBCount += s.ADBCount
			sStats.USBCount += s.USBCount
			sStats.OTGCount += s.OTGCount
		}
	}

	// Pagination
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 50
	}

	start := (page - 1) * pageSize
	end := start + pageSize

	var finalResults []status.ServerStatusStats
	if start < len(results) {
		if end > len(results) {
			end = len(results)
		}
		finalResults = results[start:end]
	} else {
		finalResults = []status.ServerStatusStats{}
	}

	sendSuccess(conn, req.ID, ServerStatusResponse{
		Items: finalResults,
		Stats: sStats,
	})
}

func handleDeviceReboot(conn *SafeConn, req Request) {
	var params DeviceControlParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Invalid params: %v", err))
		return
	}

	opts := params.ToSelectorOptions()
	devices, err := selector.SelectDevices(opts)
	if err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Error selecting devices: %v", err))
		return
	}

	if len(devices) == 0 {
		sendError(conn, req.ID, "No devices found matching criteria")
		return
	}

	cfg, err := config.Load()
	if err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Failed to load config: %v", err))
		return
	}

	ctrl := controller.NewDeviceController(cfg)
	if err := ctrl.RebootBatch(devices, func(curr, total int, msg string) {
		sendProgress(conn, req.ID, curr, total, msg)
	}); err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Reboot failed (partial or full): %v", err))
		return
	}

	sendSuccess(conn, req.ID, fmt.Sprintf("Successfully initiated reboot for %d devices", len(devices)))
}

func handleDeviceUSB(conn *SafeConn, req Request) {
	var params DeviceControlParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Invalid params: %v", err))
		return
	}

	otg := false
	mode := strings.ToLower(params.Mode)
	switch mode {
	case "host", "otg":
		otg = true
	case "device", "usb":
		otg = false
	default:
		sendError(conn, req.ID, fmt.Sprintf("Invalid mode: %s (use 'host' or 'device')", params.Mode))
		return
	}

	opts := params.ToSelectorOptions()
	devices, err := selector.SelectDevices(opts)
	if err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Error selecting devices: %v", err))
		return
	}

	if len(devices) == 0 {
		sendError(conn, req.ID, "No devices found matching criteria")
		return
	}

	cfg, err := config.Load()
	if err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Failed to load config: %v", err))
		return
	}

	ctrl := controller.NewDeviceController(cfg)
	if err := ctrl.SwitchUSBBatch(devices, otg, func(curr, total int, msg string) {
		sendProgress(conn, req.ID, curr, total, msg)
	}); err != nil {
		sendError(conn, req.ID, fmt.Sprintf("USB switch failed (partial or full): %v", err))
		return
	}

	modeStr := "USB (Device)"
	if otg {
		modeStr = "OTG (Host)"
	}
	sendSuccess(conn, req.ID, fmt.Sprintf("Successfully switched %d devices to %s", len(devices), modeStr))
}

func handleDeviceADB(conn *SafeConn, req Request) {
	var params DeviceControlParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Invalid params: %v", err))
		return
	}

	enable := false
	state := strings.ToLower(params.State)
	switch state {
	case "on", "true":
		enable = true
	case "off", "false":
		enable = false
	default:
		sendError(conn, req.ID, fmt.Sprintf("Invalid state: %s (use 'on' or 'off')", params.State))
		return
	}

	opts := params.ToSelectorOptions()
	devices, err := selector.SelectDevices(opts)
	if err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Error selecting devices: %v", err))
		return
	}

	if len(devices) == 0 {
		sendError(conn, req.ID, "No devices found matching criteria")
		return
	}

	cfg, err := config.Load()
	if err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Failed to load config: %v", err))
		return
	}

	ctrl := controller.NewDeviceController(cfg)
	if err := ctrl.ControlADBBatch(devices, enable, func(curr, total int, msg string) {
		sendProgress(conn, req.ID, curr, total, msg)
	}); err != nil {
		sendError(conn, req.ID, fmt.Sprintf("ADB control failed (partial or full): %v", err))
		return
	}

	actionStr := "disabled"
	if enable {
		actionStr = "enabled"
	}
	sendSuccess(conn, req.ID, fmt.Sprintf("Successfully %s ADB for %d devices", actionStr, len(devices)))
}

func handleConfigList(conn *SafeConn, req Request) {
	cfg := config.LoadSettings()
	if cfg == nil {
		sendError(conn, req.ID, "Failed to load config")
		return
	}

	sendSuccess(conn, req.ID, cfg)
}

type ConfigKeyParams struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func handleConfigGet(conn *SafeConn, req Request) {
	var params ConfigKeyParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		sendError(conn, req.ID, "Invalid params")
		return
	}

	cfg := config.LoadSettings()
	if cfg == nil {
		sendError(conn, req.ID, "Failed to load config")
		return
	}

	val, found := config.GetField(cfg, params.Key)
	if !found {
		sendError(conn, req.ID, fmt.Sprintf("Config key not found: %s", params.Key))
		return
	}
	sendSuccess(conn, req.ID, val)
}

func handleConfigSet(conn *SafeConn, req Request) {
	var params ConfigKeyParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		sendError(conn, req.ID, "Invalid params")
		return
	}

	cfg := config.LoadSettings()
	if cfg == nil {
		sendError(conn, req.ID, "Failed to load config")
		return
	}

	if err := config.SetField(cfg, params.Key, params.Value); err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Failed to set config: %v", err))
		return
	}

	if err := config.SaveSettings(cfg); err != nil {
		sendError(conn, req.ID, fmt.Sprintf("Failed to save config: %v", err))
		return
	}

	sendSuccess(conn, req.ID, "Config updated")
}

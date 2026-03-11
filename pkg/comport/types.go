package comport

// DeviceChannel 单通道信息（供 JSON 输出）
type DeviceChannel struct {
	Channel  int    `json:"channel"`
	Plug     string `json:"plug"` // "无主板" / "有主板" / "正在拔出" / "正在接入"
	Mode     string `json:"mode"` // "OFF" / "HUB" / "OTG"
	ModeCode byte   `json:"mode_code"`
}

// DeviceListResult 设备列表结果
type DeviceListResult struct {
	Port     string          `json:"port"`
	UID      string          `json:"uid"`
	Version  string          `json:"version"`
	MAC      string          `json:"mac"`
	IP       string          `json:"ip"`
	Mask     string          `json:"mask"`
	Gateway  string          `json:"gateway"`
	FanMode  string          `json:"fan_mode"`
	Channels []DeviceChannel `json:"channels"`
}

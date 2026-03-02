package model

// BusinessFunction codes
const (
	FuncDeviceList   = 5
	FuncSystemSync   = 111
	FuncOnlineStatus = 6

	// Device Control (Guard)
	FuncSwitchUSBGuard = 106
	FuncPowerControl   = 107
	FuncEnableADB      = 109

	// Terminal
	FuncTerminalInit = 9

	// Device Control (Mirror)
	FuncRebootDeviceMirror = 155
	FuncSwitchUSBMirror    = 218
	FuncControlADBMirror   = 219

	// Cluster/System Info
	FuncGetSystemVersion = 110
	FuncGetNetworkInfo   = 112
)

// ServerStatus represents the aggregated status of a server
type ServerStatus struct {
	Address     string           `json:"address"`
	Online      bool             `json:"online"`
	License     LicenseInfo      `json:"license"`
	DeviceCount int              `json:"device_count"`
	Devices     []DeviceListItem `json:"devices"`
	Error       string           `json:"error,omitempty"`
}

// WSRequest represents a WebSocket protocol request
type WSRequest struct {
	F    int         `msgpack:"f" json:"f"`
	Req  bool        `msgpack:"req" json:"req"`
	Seq  int         `msgpack:"seq" json:"seq"`
	Data interface{} `msgpack:"data,omitempty" json:"data,omitempty"`
	Code int         `msgpack:"code,omitempty" json:"code,omitempty"`
	Msg  string      `msgpack:"msg,omitempty" json:"msg,omitempty"`
	T    int64       `msgpack:"t,omitempty" json:"t,omitempty"`
}

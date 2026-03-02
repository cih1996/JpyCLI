package model

// DeviceInfo represents the aggregated status of a device from a middleware server
type DeviceInfo struct {
	ServerURL   string
	Seat        int
	UUID        string
	Model       string
	Android     string
	IsOnline    bool
	BizOnline   bool
	IP          string
	ADBEnabled  bool
	USBMode     bool // true = USB, false = OTG
	ServerIndex int
}

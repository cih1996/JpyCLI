package device

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"jpy-cli/pkg/middleware/device/controller"
	"jpy-cli/pkg/middleware/model"
)

// --- Shared Helpers ---

func cleanURL(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	return url
}

// --- Device List JSON ---

type DeviceListJSONOutput struct {
	Total   int                `json:"total"`
	Devices []DeviceJSONOutput `json:"devices"`
}

type DeviceJSONOutput struct {
	Server    string `json:"server"`
	Seat      int    `json:"seat"`
	UUID      string `json:"uuid"`
	IP        string `json:"ip"`
	Online    bool   `json:"online"`
	BizOnline bool   `json:"biz_online"`
	USBMode   string `json:"usb_mode"`
	ADB       bool   `json:"adb"`
	Model     string `json:"model,omitempty"`
	Android   string `json:"android,omitempty"`
}

func printDeviceListJSON(devices []model.DeviceInfo) {
	out := DeviceListJSONOutput{
		Total:   len(devices),
		Devices: make([]DeviceJSONOutput, len(devices)),
	}
	for i, d := range devices {
		usbMode := "otg"
		if d.USBMode {
			usbMode = "usb"
		}
		out.Devices[i] = DeviceJSONOutput{
			Server:    cleanURL(d.ServerURL),
			Seat:      d.Seat,
			UUID:      d.UUID,
			IP:        d.IP,
			Online:    d.IsOnline,
			BizOnline: d.BizOnline,
			USBMode:   usbMode,
			ADB:       d.ADBEnabled,
			Model:     d.Model,
			Android:   d.Android,
		}
	}
	data, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(data))
}

func printDeviceListPlain(devices []model.DeviceInfo) {
	fmt.Println("SERVER\tSEAT\tUUID\tMODEL\tANDROID\tONLINE\tBIZ\tIP\tADB\tUSB")
	for _, d := range devices {
		usbMode := "otg"
		if d.USBMode {
			usbMode = "usb"
		}
		fmt.Printf("%s\t%d\t%s\t%s\t%s\t%v\t%v\t%s\t%v\t%s\n",
			cleanURL(d.ServerURL), d.Seat, d.UUID, d.Model, d.Android,
			d.IsOnline, d.BizOnline, d.IP, d.ADBEnabled, usbMode)
	}
}

// --- Status JSON ---

type StatusJSONOutput struct {
	Summary StatusSummaryJSON          `json:"summary"`
	Servers []ServerStatusJSONOutput   `json:"servers"`
}

type StatusSummaryJSON struct {
	TotalServers  int `json:"total_servers"`
	OnlineServers int `json:"online_servers"`
	TotalDevices  int `json:"total_devices"`
	BizOnline     int `json:"biz_online"`
	IPCount       int `json:"ip_count"`
	UUIDCount     int `json:"uuid_count"`
	ADBCount      int `json:"adb_count"`
	USBCount      int `json:"usb_count"`
	OTGCount      int `json:"otg_count"`
}

type ServerStatusJSONOutput struct {
	Address         string `json:"address"`
	Status          string `json:"status"`
	Authorized      bool   `json:"authorized"`
	LicenseStatus   string `json:"license_status"`
	SN              string `json:"sn,omitempty"`
	ControlAddr     string `json:"control_addr,omitempty"`
	LicenseName     string `json:"license_name,omitempty"`
	FirmwareVersion string `json:"firmware_version,omitempty"`
	NetworkSpeed    string `json:"network_speed,omitempty"`
	DeviceCount     int    `json:"device_count"`
	BizOnlineCount  int    `json:"biz_online_count"`
	IPCount         int    `json:"ip_count"`
	UUIDCount       int    `json:"uuid_count"`
	ADBCount        int    `json:"adb_count"`
	USBCount        int    `json:"usb_count"`
	OTGCount        int    `json:"otg_count"`
}

func printStatusJSON(results []ServerStatusStats) {
	out := StatusJSONOutput{
		Servers: make([]ServerStatusJSONOutput, len(results)),
	}
	for i, r := range results {
		out.Summary.TotalDevices += r.DeviceCount
		out.Summary.BizOnline += r.BizOnlineCount
		out.Summary.IPCount += r.IPCount
		out.Summary.UUIDCount += r.UUIDCount
		out.Summary.ADBCount += r.ADBCount
		out.Summary.USBCount += r.USBCount
		out.Summary.OTGCount += r.OTGCount
		if r.Status == "Online" {
			out.Summary.OnlineServers++
		}
		out.Servers[i] = ServerStatusJSONOutput{
			Address:         cleanURL(r.ServerURL),
			Status:          r.Status,
			Authorized:      r.LicenseStatus == "成功",
			LicenseStatus:   r.LicenseStatus,
			SN:              r.SN,
			ControlAddr:     r.ControlAddr,
			LicenseName:     r.LicenseName,
			FirmwareVersion: r.FirmwareVersion,
			NetworkSpeed:    r.NetworkSpeed,
			DeviceCount:     r.DeviceCount,
			BizOnlineCount:  r.BizOnlineCount,
			IPCount:         r.IPCount,
			UUIDCount:       r.UUIDCount,
			ADBCount:        r.ADBCount,
			USBCount:        r.USBCount,
			OTGCount:        r.OTGCount,
		}
	}
	out.Summary.TotalServers = len(results)
	data, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(data))
}

func printStatusPlain(results []ServerStatusStats, detail bool) {
	if detail {
		fmt.Println("SERVER\tSTATUS\tFIRMWARE\tSPEED\tDEVICES\tBIZ_ONLINE\tIP\tUUID\tADB\tOTG\tUSB\tAUTH\tSN\tCONTROL\tNAME")
	} else {
		fmt.Println("SERVER\tSTATUS\tDEVICES\tBIZ_ONLINE\tIP\tUUID\tADB\tOTG\tUSB\tAUTH")
	}

	var totalDevice, totalBiz, totalIP, totalUUID, totalADB, totalUSB, totalOTG int
	for _, r := range results {
		totalDevice += r.DeviceCount
		totalBiz += r.BizOnlineCount
		totalIP += r.IPCount
		totalUUID += r.UUIDCount
		totalADB += r.ADBCount
		totalUSB += r.USBCount
		totalOTG += r.OTGCount

		if detail {
			fmt.Printf("%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%s\t%s\t%s\t%s\n",
				cleanURL(r.ServerURL), r.Status,
				r.FirmwareVersion, r.NetworkSpeed,
				r.DeviceCount, r.BizOnlineCount, r.IPCount, r.UUIDCount, r.ADBCount,
				r.OTGCount, r.USBCount,
				r.LicenseStatus, r.SN, r.ControlAddr, r.LicenseName)
		} else {
			fmt.Printf("%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%s\n",
				cleanURL(r.ServerURL), r.Status,
				r.DeviceCount, r.BizOnlineCount, r.IPCount, r.UUIDCount, r.ADBCount,
				r.OTGCount, r.USBCount,
				r.LicenseStatus)
		}
	}
	fmt.Printf("---\nTOTAL(%d servers)\t-\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t-\n",
		len(results), totalDevice, totalBiz, totalIP, totalUUID, totalADB, totalOTG, totalUSB)
}

// --- Control Command Output ---

type ControlJSONOutput struct {
	Total   int                 `json:"total"`
	Success int                 `json:"success"`
	Failed  int                 `json:"failed"`
	Results []ControlResultJSON `json:"results"`
}

type ControlResultJSON struct {
	Server string `json:"server"`
	Seat   int    `json:"seat"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

func printControlJSON(results []controller.BatchResult) {
	out := ControlJSONOutput{
		Total:   len(results),
		Results: make([]ControlResultJSON, len(results)),
	}
	for i, r := range results {
		status := "ok"
		errStr := ""
		if !r.OK {
			status = "failed"
			errStr = r.Error
			out.Failed++
		} else {
			out.Success++
		}
		out.Results[i] = ControlResultJSON{
			Server: cleanURL(r.Server),
			Seat:   r.Seat,
			Status: status,
			Error:  errStr,
		}
	}
	data, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(data))
}

func printControlPlain(results []controller.BatchResult) {
	fmt.Println("SERVER\tSEAT\tSTATUS\tERROR")
	success := 0
	for _, r := range results {
		status := "ok"
		errStr := ""
		if !r.OK {
			status = "failed"
			errStr = r.Error
		} else {
			success++
		}
		fmt.Printf("%s\t%d\t%s\t%s\n", cleanURL(r.Server), r.Seat, status, errStr)
	}
	fmt.Fprintf(os.Stderr, "总计: %d, 成功: %d, 失败: %d\n", len(results), success, len(results)-success)
}

// controlExitCode 根据结果返回退出码 (0=全成功, 1=部分失败, 2=全失败)
func controlExitCode(results []controller.BatchResult) int {
	if len(results) == 0 {
		return 0
	}
	success := 0
	for _, r := range results {
		if r.OK {
			success++
		}
	}
	if success == len(results) {
		return 0
	}
	if success == 0 {
		return 2
	}
	return 1
}

// --- Export Output ---

type ExportJSONOutput struct {
	Total   int                `json:"total"`
	Devices []DeviceJSONOutput `json:"devices"`
}

func printExportJSON(devices []model.DeviceInfo) {
	// 复用 DeviceListJSON 格式
	printDeviceListJSON(devices)
}

func printExportPlain(devices []model.DeviceInfo) {
	// 复用 DeviceListPlain 格式
	printDeviceListPlain(devices)
}

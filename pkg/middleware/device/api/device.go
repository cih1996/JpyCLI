package api

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"jpy-cli/pkg/middleware/model"
	"jpy-cli/pkg/middleware/protocol"
	"net/http"
	"strings"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

// DeviceAPI provides methods for interacting with device-related functionality.
type DeviceAPI struct {
	transport  protocol.Transport
	baseURL    string
	token      string
	httpClient *http.Client
}

func NewDeviceAPI(t protocol.Transport, baseURL, token string) *DeviceAPI {
	// Normalize baseURL (remove trailing slash)
	baseURL = strings.TrimSuffix(baseURL, "/")

	// Create insecure HTTP client
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   5 * time.Second,
	}

	return &DeviceAPI{
		transport:  t,
		baseURL:    baseURL,
		token:      token,
		httpClient: client,
	}
}

// FetchDeviceList retrieves the list of devices from the server.
func (api *DeviceAPI) FetchDeviceList() ([]model.DeviceListItem, error) {
	resp, err := api.transport.SendRequest(model.FuncDeviceList, nil)
	if err != nil {
		return nil, err
	}

	// Extract data from response
	b, _ := msgpack.Marshal(resp.Data)

	// Try unwrapped array from Data
	var devicesFromData []model.DeviceListItem
	decData := msgpack.NewDecoder(bytes.NewReader(b))
	decData.SetCustomStructTag("json")
	if err := decData.Decode(&devicesFromData); err == nil {
		return devicesFromData, nil
	}

	// Try wrapped {data: [...]}
	var wrapper struct {
		Data []model.DeviceListItem `json:"data"`
	}
	decWrapper := msgpack.NewDecoder(bytes.NewReader(b))
	decWrapper.SetCustomStructTag("json")
	if err := decWrapper.Decode(&wrapper); err == nil {
		return wrapper.Data, nil
	}

	return nil, errors.New("解析设备列表失败")
}

// FetchOnlineStatus retrieves the online status of devices.
func (api *DeviceAPI) FetchOnlineStatus() ([]model.OnlineStatus, error) {
	resp, err := api.transport.SendRequest(model.FuncOnlineStatus, nil)
	if err != nil {
		return nil, err
	}

	b, _ := msgpack.Marshal(resp.Data)

	// Try unwrapped array from Data
	var statusesFromData []model.OnlineStatus
	decData := msgpack.NewDecoder(bytes.NewReader(b))
	decData.SetCustomStructTag("json")
	if err := decData.Decode(&statusesFromData); err == nil {
		return statusesFromData, nil
	}

	// Try wrapped {data: [...]}
	var wrapper struct {
		Data []model.OnlineStatus `json:"data"`
	}
	decWrapper := msgpack.NewDecoder(bytes.NewReader(b))
	decWrapper.SetCustomStructTag("json")
	if err := decWrapper.Decode(&wrapper); err == nil {
		return wrapper.Data, nil
	}

	return nil, errors.New("解析在线状态失败")
}

// RebootDevice reboots the device.
func (api *DeviceAPI) RebootDevice(seat int) error {
	return api.sendControlRequest(model.FuncPowerControl, map[string]interface{}{
		"seat": seat,
		"mode": 2, // 2=Reboot
	})
}

// SwitchUSBMode switches the USB mode (true for Host/OTG, false for Device/USB).
func (api *DeviceAPI) SwitchUSBMode(seat int, otg bool) error {
	mode := 1 // USB
	if otg {
		mode = 0 // OTG
	}
	return api.sendControlRequest(model.FuncSwitchUSBGuard, map[string]interface{}{
		"seat": seat,
		"mode": mode,
	})
}

// ControlADB enables or disables ADB.
func (api *DeviceAPI) ControlADB(seat int, enable bool) error {
	mode := 0
	if enable {
		mode = 2
	}
	return api.sendControlRequest(model.FuncEnableADB, map[string]interface{}{
		"seat": seat,
		"mode": mode,
	})
}

// sendControlRequest sends a control request and checks for server errors.
func (api *DeviceAPI) sendControlRequest(code int, data interface{}) error {
	resp, err := api.transport.SendRequest(code, data)
	if err != nil {
		return err
	}
	if resp.Code != nil && *resp.Code != 0 {
		msg := "unknown error"
		if resp.Msg != nil {
			msg = *resp.Msg
		}
		return fmt.Errorf("operation failed (code %d): %s", *resp.Code, msg)
	}
	return nil
}

// GetSystemVersion retrieves the system version information via HTTP API.
func (api *DeviceAPI) GetSystemVersion() (*model.SystemVersion, error) {
	if api.baseURL == "" {
		return nil, errors.New("baseURL not configured")
	}

	url := fmt.Sprintf("%s/sys/version", api.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", api.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Code int                  `json:"code"`
		Data *model.SystemVersion `json:"data"`
		Msg  string               `json:"msg"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}

	if result.Code != 200 {
		return nil, fmt.Errorf("api error: %s", result.Msg)
	}

	if result.Data == nil {
		// If data is null but success, return empty struct or error?
		// TS implementation returns default values if data exists.
		return &model.SystemVersion{Version: "-"}, nil
	}

	// Ensure version is set (TS logic: result.data.version || '-')
	if result.Data.Version == "" {
		result.Data.Version = "-"
	}

	return result.Data, nil
}

// GetNetworkInfo retrieves the network information via HTTP API.
func (api *DeviceAPI) GetNetworkInfo() (*model.NetworkInfo, error) {
	if api.baseURL == "" {
		return nil, errors.New("baseURL not configured")
	}

	url := fmt.Sprintf("%s/sys/network", api.baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", api.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Raw response structure
	var result struct {
		Code int `json:"code"`
		Data []struct {
			Speed interface{} `json:"Speed"`
			IPv4  struct {
				Addresses []struct {
					Address string `json:"Address"`
				} `json:"Addresses"`
			} `json:"IPv4"`
		} `json:"data"`
		Msg string `json:"msg"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}

	if result.Code != 200 {
		return nil, fmt.Errorf("api error: %s", result.Msg)
	}

	if len(result.Data) == 0 {
		return nil, errors.New("no network data")
	}

	netData := result.Data[0]

	// Convert Speed to float64 pointer for IL struct
	var speedVal float64
	switch v := netData.Speed.(type) {
	case float64:
		speedVal = v
	case int:
		speedVal = float64(v)
	default:
		speedVal = 0
	}

	speedIL := &model.IL{Double: &speedVal}

	// Extract IPv4
	var ipv4Str string
	if len(netData.IPv4.Addresses) > 0 {
		ipv4Str = netData.IPv4.Addresses[0].Address
	}

	return &model.NetworkInfo{
		Speed: speedIL,
		IPv4:  &ipv4Str,
	}, nil
}

// RestartService restarts a specific service on the device.
func (api *DeviceAPI) RestartService(service string, action int) error {
	if api.baseURL == "" {
		return errors.New("baseURL not configured")
	}

	url := fmt.Sprintf("%s/sys/service", api.baseURL)
	payload := map[string]interface{}{
		"service": service,
		"action":  action,
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", api.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := api.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decode error: %v", err)
	}

	if result.Code != 200 {
		return fmt.Errorf("api error: %s", result.Msg)
	}

	return nil
}

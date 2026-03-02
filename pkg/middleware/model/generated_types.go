// Code generated from JSON Schema using quicktype. DO NOT EDIT.
// To parse and unparse this JSON data, add this code to your project and do:
//
//    deviceListItem, err := UnmarshalDeviceListItem(bytes)
//    bytes, err = deviceListItem.Marshal()
//
//    cAPTCHAData, err := UnmarshalCAPTCHAData(bytes)
//    bytes, err = cAPTCHAData.Marshal()
//
//    licenseData, err := UnmarshalLicenseData(bytes)
//    bytes, err = licenseData.Marshal()
//
//    loginResult, err := UnmarshalLoginResult(bytes)
//    bytes, err = loginResult.Marshal()
//
//    onlineStatus, err := UnmarshalOnlineStatus(bytes)
//    bytes, err = onlineStatus.Marshal()
//
//    shellResult, err := UnmarshalShellResult(bytes)
//    bytes, err = shellResult.Marshal()
//
//    appInfo, err := UnmarshalAppInfo(bytes)
//    bytes, err = appInfo.Marshal()
//
//    deviceDetail, err := UnmarshalDeviceDetail(bytes)
//    bytes, err = deviceDetail.Marshal()
//
//    licenseInfo, err := UnmarshalLicenseInfo(bytes)
//    bytes, err = licenseInfo.Marshal()
//
//    networkInfo, err := UnmarshalNetworkInfo(bytes)
//    bytes, err = networkInfo.Marshal()
//
//    mirrorWSConfig, err := UnmarshalMirrorWSConfig(bytes)
//    bytes, err = mirrorWSConfig.Marshal()
//
//    guardWSConfig, err := UnmarshalGuardWSConfig(bytes)
//    bytes, err = guardWSConfig.Marshal()
//
//    rOMPackage, err := UnmarshalROMPackage(bytes)
//    bytes, err = rOMPackage.Marshal()
//
//    rOMFlashProgressData, err := UnmarshalROMFlashProgressData(bytes)
//    bytes, err = rOMFlashProgressData.Marshal()
//
//    batchOperationResult, err := UnmarshalBatchOperationResult(bytes)
//    bytes, err = batchOperationResult.Marshal()
//
//    flashTaskStatus, err := UnmarshalFlashTaskStatus(bytes)
//    bytes, err = flashTaskStatus.Marshal()
//
//    commandResponse, err := UnmarshalCommandResponse(bytes)
//    bytes, err = commandResponse.Marshal()
//
//    downloadTask, err := UnmarshalDownloadTask(bytes)
//    bytes, err = downloadTask.Marshal()
//
//    uIElement, err := UnmarshalUIElement(bytes)
//    bytes, err = uIElement.Marshal()
//
//    wSBaseMessage, err := UnmarshalWSBaseMessage(bytes)
//    bytes, err = wSBaseMessage.Marshal()
//
//    wSResponse, err := UnmarshalWSResponse(bytes)
//    bytes, err = wSResponse.Marshal()
//
//    wSPushMessage, err := UnmarshalWSPushMessage(bytes)
//    bytes, err = wSPushMessage.Marshal()
//
//    middlewareClientConfig, err := UnmarshalMiddlewareClientConfig(bytes)
//    bytes, err = middlewareClientConfig.Marshal()
//
//    keyAction, err := UnmarshalKeyAction(bytes)
//    bytes, err = keyAction.Marshal()
//
//    touchAction, err := UnmarshalTouchAction(bytes)
//    bytes, err = touchAction.Marshal()
//
//    connectionStatus, err := UnmarshalConnectionStatus(bytes)
//    bytes, err = connectionStatus.Marshal()
//
//    clusterWSCallbacks, err := UnmarshalClusterWSCallbacks(bytes)
//    bytes, err = clusterWSCallbacks.Marshal()
//
//    remoteServerConfig, err := UnmarshalRemoteServerConfig(bytes)
//    bytes, err = remoteServerConfig.Marshal()

package model

import (
	"bytes"
	"encoding/json"
	"errors"
)

func UnmarshalDeviceListItem(data []byte) (DeviceListItem, error) {
	var r DeviceListItem
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *DeviceListItem) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalCAPTCHAData(data []byte) (CAPTCHAData, error) {
	var r CAPTCHAData
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *CAPTCHAData) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalLicenseData(data []byte) (LicenseData, error) {
	var r LicenseData
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *LicenseData) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalLoginResult(data []byte) (LoginResult, error) {
	var r LoginResult
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *LoginResult) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalOnlineStatus(data []byte) (OnlineStatus, error) {
	var r OnlineStatus
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *OnlineStatus) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalShellResult(data []byte) (ShellResult, error) {
	var r ShellResult
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ShellResult) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalAppInfo(data []byte) (AppInfo, error) {
	var r AppInfo
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *AppInfo) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalDeviceDetail(data []byte) (DeviceDetail, error) {
	var r DeviceDetail
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *DeviceDetail) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalLicenseInfo(data []byte) (LicenseInfo, error) {
	var r LicenseInfo
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *LicenseInfo) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalSystemVersion(data []byte) (SystemVersion, error) {
	var r SystemVersion
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *SystemVersion) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalNetworkInfo(data []byte) (NetworkInfo, error) {
	var r NetworkInfo
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *NetworkInfo) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalMirrorWSConfig(data []byte) (MirrorWSConfig, error) {
	var r MirrorWSConfig
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *MirrorWSConfig) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalGuardWSConfig(data []byte) (GuardWSConfig, error) {
	var r GuardWSConfig
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *GuardWSConfig) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalROMPackage(data []byte) (ROMPackage, error) {
	var r ROMPackage
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ROMPackage) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalROMFlashProgressData(data []byte) (ROMFlashProgressData, error) {
	var r ROMFlashProgressData
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ROMFlashProgressData) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalBatchOperationResult(data []byte) (BatchOperationResult, error) {
	var r BatchOperationResult
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *BatchOperationResult) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalFlashTaskStatus(data []byte) (FlashTaskStatus, error) {
	var r FlashTaskStatus
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *FlashTaskStatus) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalCommandResponse(data []byte) (CommandResponse, error) {
	var r CommandResponse
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *CommandResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalDownloadTask(data []byte) (DownloadTask, error) {
	var r DownloadTask
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *DownloadTask) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalUIElement(data []byte) (UIElement, error) {
	var r UIElement
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *UIElement) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalWSBaseMessage(data []byte) (WSBaseMessage, error) {
	var r WSBaseMessage
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *WSBaseMessage) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalWSResponse(data []byte) (WSResponse, error) {
	var r WSResponse
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *WSResponse) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalWSPushMessage(data []byte) (WSPushMessage, error) {
	var r WSPushMessage
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *WSPushMessage) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalMiddlewareClientConfig(data []byte) (MiddlewareClientConfig, error) {
	var r MiddlewareClientConfig
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *MiddlewareClientConfig) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalKeyAction(data []byte) (KeyAction, error) {
	var r KeyAction
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *KeyAction) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalTouchAction(data []byte) (TouchAction, error) {
	var r TouchAction
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *TouchAction) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalConnectionStatus(data []byte) (ConnectionStatus, error) {
	var r ConnectionStatus
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ConnectionStatus) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalClusterWSCallbacks(data []byte) (ClusterWSCallbacks, error) {
	var r ClusterWSCallbacks
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *ClusterWSCallbacks) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

func UnmarshalRemoteServerConfig(data []byte) (RemoteServerConfig, error) {
	var r RemoteServerConfig
	err := json.Unmarshal(data, &r)
	return r, err
}

func (r *RemoteServerConfig) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// 设备列表项（f=5 响应）
type DeviceListItem struct {
	AndroidVersion *string     `json:"androidVersion,omitempty"`
	CPU            int         `json:"cpu"`
	DiskSize       interface{} `json:"diskSize"`
	Height         int         `json:"height"`
	Memory         interface{} `json:"memory"`
	Model          string      `json:"model"`
	OSVersion      string      `json:"osVersion"`
	Seat           int         `json:"seat"`
	Type           int         `json:"type"`
	UUID           string      `json:"uuid"`
	Width          int         `json:"width"`
}

// 验证码数据
type CAPTCHAData struct {
	ID    string `json:"id"`
	Image string `json:"image"`
}

// 授权信息数据
type LicenseData struct {
	B         float64 `json:"B"`
	C         string  `json:"C"`
	CT        float64 `json:"CT"`
	DN        string  `json:"DN"`
	H         string  `json:"H"`
	I         float64 `json:"I"`
	IL        *IL     `json:"IL"`
	L         float64 `json:"L"`
	M         float64 `json:"M"`
	N         string  `json:"N"`
	S         bool    `json:"S"`
	Sn        string  `json:"SN"`
	Status    int     `json:"status"`
	StatusTxt string  `json:"statusTxt"`
	T         float64 `json:"T"`
	Used      float64 `json:"used"`
	VT        float64 `json:"VT"`
}

// 登录结果
type LoginResult struct {
	Error   *string `json:"error,omitempty"`
	Success bool    `json:"success"`
	Token   *string `json:"token,omitempty"`
}

// 设备在线状态详细信息 (f=6)
type OnlineStatus struct {
	IP                   string  `json:"ip"`
	IsADBEnabled         bool    `json:"isADBEnabled"`
	IsBusinessOnline     bool    `json:"isBusinessOnline"`
	IsControlBoardOnline bool    `json:"isControlBoardOnline"`
	IsManagementOnline   bool    `json:"isManagementOnline"`
	IsUSBMode            bool    `json:"isUSBMode"`
	Online               float64 `json:"online"`
	Seat                 int     `json:"seat"`
	Time                 *IL     `json:"time"`
}

// Shell 执行结果 (f=289)
type ShellResult struct {
	Error    *string `json:"error,omitempty"`
	ExitCode *int    `json:"exitCode,omitempty"`
	Output   string  `json:"output"`
}

// App 信息 (f=290)
type AppInfo struct {
	AppName     string  `json:"appName"`
	InstallTime *IL     `json:"installTime"`
	PackageName string  `json:"packageName"`
	Size        *IL     `json:"size"`
	VersionCode *int    `json:"versionCode,omitempty"`
	VersionName *string `json:"versionName,omitempty"`
}

// 设备详细信息 (f=4)
// 包含设备的完整硬件、系统、屏幕、语言等信息
type DeviceDetail struct {
	AndroidVersion *string  `json:"androidVersion,omitempty"`
	AppVersion     *string  `json:"appVersion,omitempty"`
	Brand          string   `json:"brand"`
	Country        string   `json:"country"`
	CPU            int      `json:"cpu"`
	DiskSize       *IL      `json:"diskSize"`
	Height         int      `json:"height"`
	Lang           string   `json:"lang"`
	Location       *string  `json:"location,omitempty"`
	Memory         *IL      `json:"memory"`
	Model          string   `json:"model"`
	Orientation    float64  `json:"orientation"`
	OS             string   `json:"os"`
	OSVersion      string   `json:"osVersion"`
	Seat           int      `json:"seat"`
	Self           *string  `json:"self,omitempty"`
	SignalMode     *string  `json:"signalMode,omitempty"`
	SysPer         *float64 `json:"sysPer,omitempty"`
	SysVersion     *string  `json:"sysVersion,omitempty"`
	Timezone       string   `json:"timezone"`
	Type           int      `json:"type"`
	UUID           string   `json:"uuid"`
	Vendor         *string  `json:"vendor,omitempty"`
	Width          int      `json:"width"`
}

// 授权信息
type LicenseInfo struct {
	ExpireTime *IL     `json:"expireTime"`
	MaxDevices int     `json:"maxDevices"`
	Status     *int    `json:"status,omitempty"`
	StatusTxt  *string `json:"statusTxt,omitempty"`
	Valid      bool    `json:"valid"`
}

// 系统版本信息
type SystemVersion struct {
	Version   string  `json:"version"`
	Project   *string `json:"project,omitempty"`
	Timestamp *int64  `json:"timestamp,omitempty"`
	Arch      *string `json:"arch,omitempty"`
	OS        *string `json:"os,omitempty"`
	Info      *string `json:"info,omitempty"`
}

// 网络信息
type NetworkInfo struct {
	DNS            *string `json:"dns,omitempty"`
	Gateway        *string `json:"gateway,omitempty"`
	Interface      *string `json:"Interface,omitempty"`
	IP             *string `json:"ip,omitempty"`
	IPv4           *string `json:"IPv4,omitempty"`
	NetworkInfoMAC *string `json:"mac,omitempty"`
	MAC            *string `json:"MAC,omitempty"`
	Mask           *string `json:"mask,omitempty"`
	Speed          *IL     `json:"Speed"`
}

// Mirror WebSocket 配置
type MirrorWSConfig struct {
	AutoReconnect        *bool    `json:"autoReconnect,omitempty"`
	DeviceID             float64  `json:"deviceId"`
	MaxReconnectAttempts *float64 `json:"maxReconnectAttempts,omitempty"`
	ReconnectInterval    *float64 `json:"reconnectInterval,omitempty"`
	Token                string   `json:"token"`
	URL                  string   `json:"url"`
}

// Guard WebSocket 配置
type GuardWSConfig struct {
	AutoReconnect        *bool    `json:"autoReconnect,omitempty"`
	DeviceID             float64  `json:"deviceId"`
	MaxReconnectAttempts *float64 `json:"maxReconnectAttempts,omitempty"`
	ReconnectInterval    *float64 `json:"reconnectInterval,omitempty"`
	Token                string   `json:"token"`
	URL                  string   `json:"url"`
}

// ROM 包信息
type ROMPackage struct {
	Desc    string `json:"desc"`
	Model   string `json:"model"`
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ROM 刷入进度数据 (f=118)
type ROMFlashProgressData struct {
	EndTime   *IL     `json:"endTime"`
	LastError *string `json:"lastError,omitempty"`
	Message   string  `json:"message"`
	Progress  float64 `json:"progress"`
	Seat      int     `json:"seat"`
	StartTime *IL     `json:"startTime"`
	Status    int     `json:"status"`
	Step      string  `json:"step"`
}

// 批量操作结果
type BatchOperationResult struct {
	Error   interface{} `json:"error"`
	Seat    int         `json:"seat"`
	Success bool        `json:"success"`
}

// 刷机任务状态
type FlashTaskStatus struct {
	DeviceID  float64 `json:"deviceId"`
	EndTime   *IL     `json:"endTime"`
	Image     string  `json:"image"`
	LastError string  `json:"lastError"`
	Progress  float64 `json:"progress"`
	QueueTime *IL     `json:"queueTime"`
	Session   string  `json:"session"`
	StartTime *IL     `json:"startTime"`
	Status    int     `json:"status"`
	TaskID    string  `json:"taskId"`
}

// 通用命令执行响应
//
// 用于返回结构未明确定义的命令响应。
// 当看到方法返回此类型且标记了 TODO 时，表示需要补充具体的返回结构。
//
// 通过控制台日志 [SDK-Response:*] 查看实际返回数据，然后补充具体的接口定义。
type CommandResponse struct {
	Data    interface{} `json:"data"`
	Error   *string     `json:"error,omitempty"`
	Success bool        `json:"success"`
}

// 下载任务信息 (f=296)
type DownloadTask struct {
	DownloadedSize *IL     `json:"downloadedSize"`
	Error          *string `json:"error,omitempty"`
	Name           string  `json:"name"`
	Progress       float64 `json:"progress"`
	Sha256         *string `json:"sha256,omitempty"`
	Speed          float64 `json:"speed"`
	Status         int     `json:"status"`
	TaskID         string  `json:"taskId"`
	TotalSize      *IL     `json:"totalSize"`
	URL            string  `json:"url"`
}

// 界面元素信息 (f=321)
type UIElement struct {
	Bounds     map[string]float64 `json:"bounds"`
	Children   []UIElement        `json:"children,omitempty"`
	Class      string             `json:"class"`
	Clickable  bool               `json:"clickable"`
	Depth      float64            `json:"depth"`
	Desc       string             `json:"desc"`
	Editable   bool               `json:"editable"`
	Enabled    bool               `json:"enabled"`
	Focused    bool               `json:"focused"`
	ID         string             `json:"id"`
	Index      float64            `json:"index"`
	Package    string             `json:"package"`
	Scrollable bool               `json:"scrollable"`
	Selected   bool               `json:"selected"`
	Text       string             `json:"text"`
}

// 基础 WebSocket 消息体
type WSBaseMessage struct {
	Code *int        `json:"code,omitempty"`
	Data interface{} `json:"data"`
	F    *int        `json:"f,omitempty"`
	Msg  *string     `json:"msg,omitempty"`
	Req  *bool       `json:"req,omitempty"`
	Seq  *int        `json:"seq,omitempty"`
}

// WebSocket 响应消息
type WSResponse struct {
	Code *int        `json:"code,omitempty"`
	Data interface{} `json:"data"`
	F    *int        `json:"f,omitempty"`
	Msg  *string     `json:"msg,omitempty"`
	Req  *bool       `json:"req,omitempty"`
	Seq  *int        `json:"seq,omitempty"`
}

// WebSocket 推送消息
type WSPushMessage struct {
	Code *int        `json:"code,omitempty"`
	Data interface{} `json:"data"`
	F    int         `json:"f"`
	Msg  *string     `json:"msg,omitempty"`
	Req  bool        `json:"req"`
	Seq  *int        `json:"seq,omitempty"`
}

// 中间件客户端配置
type MiddlewareClientConfig struct {
	APIBase string  `json:"apiBase"`
	Token   *string `json:"token,omitempty"`
}

// 按键动作
type KeyAction struct {
	Action float64 `json:"Action"`
	Button float64 `json:"Button"`
}

// 触摸动作
type TouchAction struct {
	D *float64 `json:"d,omitempty"`
	I *float64 `json:"i,omitempty"`
	T float64  `json:"t"`
	X float64  `json:"x"`
	Y float64  `json:"y"`
}

// Cluster WebSocket 回调接口
type ClusterWSCallbacks struct {
	OnDeviceListUpdate        *OnDeviceListUpdate        `json:"onDeviceListUpdate,omitempty"`
	OnDeviceOnlineUpdate      *OnDeviceOnlineUpdate      `json:"onDeviceOnlineUpdate,omitempty"`
	OnLicenseInfoUpdate       *OnLicenseInfoUpdate       `json:"onLicenseInfoUpdate,omitempty"`
	OnNetworkInfoUpdate       *OnNetworkInfoUpdate       `json:"onNetworkInfoUpdate,omitempty"`
	OnOperationStatusChange   *OnOperationStatusChange   `json:"onOperationStatusChange,omitempty"`
	OnReconnectAttemptsChange *OnReconnectAttemptsChange `json:"onReconnectAttemptsChange,omitempty"`
	OnStatusChange            *OnStatusChange            `json:"onStatusChange,omitempty"`
	OnSystemVersionUpdate     *OnSystemVersionUpdate     `json:"onSystemVersionUpdate,omitempty"`
}

type OnDeviceListUpdate struct {
}

type OnDeviceOnlineUpdate struct {
}

type OnLicenseInfoUpdate struct {
}

type OnNetworkInfoUpdate struct {
}

type OnOperationStatusChange struct {
}

type OnReconnectAttemptsChange struct {
}

type OnStatusChange struct {
}

type OnSystemVersionUpdate struct {
}

// 服务器配置
type RemoteServerConfig struct {
	ID    string  `json:"id"`
	Name  *string `json:"name,omitempty"`
	Token string  `json:"token"`
	URL   string  `json:"url"`
}

// 连接状态
type ConnectionStatus string

const (
	Connecting ConnectionStatus = "connecting"
	Offline    ConnectionStatus = "offline"
	Online     ConnectionStatus = "online"
)

type IL struct {
	Double *float64
	String *string
}

func (x *IL) UnmarshalJSON(data []byte) error {
	object, err := unmarshalUnion(data, nil, &x.Double, nil, &x.String, false, nil, false, nil, false, nil, false, nil, false)
	if err != nil {
		return err
	}
	if object {
	}
	return nil
}

func (x *IL) MarshalJSON() ([]byte, error) {
	return marshalUnion(nil, x.Double, nil, x.String, false, nil, false, nil, false, nil, false, nil, false)
}

func unmarshalUnion(data []byte, pi **int64, pf **float64, pb **bool, ps **string, haveArray bool, pa interface{}, haveObject bool, pc interface{}, haveMap bool, pm interface{}, haveEnum bool, pe interface{}, nullable bool) (bool, error) {
	if pi != nil {
		*pi = nil
	}
	if pf != nil {
		*pf = nil
	}
	if pb != nil {
		*pb = nil
	}
	if ps != nil {
		*ps = nil
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		return false, err
	}

	switch v := tok.(type) {
	case json.Number:
		if pi != nil {
			i, err := v.Int64()
			if err == nil {
				*pi = &i
				return false, nil
			}
		}
		if pf != nil {
			f, err := v.Float64()
			if err == nil {
				*pf = &f
				return false, nil
			}
			return false, errors.New("Unparsable number")
		}
		return false, errors.New("Union does not contain number")
	case float64:
		return false, errors.New("Decoder should not return float64")
	case bool:
		if pb != nil {
			*pb = &v
			return false, nil
		}
		return false, errors.New("Union does not contain bool")
	case string:
		if haveEnum {
			return false, json.Unmarshal(data, pe)
		}
		if ps != nil {
			*ps = &v
			return false, nil
		}
		return false, errors.New("Union does not contain string")
	case nil:
		if nullable {
			return false, nil
		}
		return false, errors.New("Union does not contain null")
	case json.Delim:
		if v == '{' {
			if haveObject {
				return true, json.Unmarshal(data, pc)
			}
			if haveMap {
				return false, json.Unmarshal(data, pm)
			}
			return false, errors.New("Union does not contain object")
		}
		if v == '[' {
			if haveArray {
				return false, json.Unmarshal(data, pa)
			}
			return false, errors.New("Union does not contain array")
		}
		return false, errors.New("Cannot handle delimiter")
	}
	return false, errors.New("Cannot unmarshal union")
}

func marshalUnion(pi *int64, pf *float64, pb *bool, ps *string, haveArray bool, pa interface{}, haveObject bool, pc interface{}, haveMap bool, pm interface{}, haveEnum bool, pe interface{}, nullable bool) ([]byte, error) {
	if pi != nil {
		return json.Marshal(*pi)
	}
	if pf != nil {
		return json.Marshal(*pf)
	}
	if pb != nil {
		return json.Marshal(*pb)
	}
	if ps != nil {
		return json.Marshal(*ps)
	}
	if haveArray {
		return json.Marshal(pa)
	}
	if haveObject {
		return json.Marshal(pc)
	}
	if haveMap {
		return json.Marshal(pm)
	}
	if haveEnum {
		return json.Marshal(pe)
	}
	if nullable {
		return json.Marshal(nil)
	}
	return nil, errors.New("Union must not be null")
}

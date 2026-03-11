// Package comport 实现上位机 COM 通讯协议（报文格式、CRC16、建连/读系统信息/设置模式）
package comport

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
)

// 报文头
const (
	HeaderByte0 = 0xDC
	HeaderByte1 = 0xAC
)

// 父指令
const (
	ParentNotify = 0x00 // 设备通知
	ParentSet    = 0x01 // 设备设置
)

// 子指令
const (
	SubConnect       = 0x02 // 建立连接（原 0x04 改为 0x02，对齐实际抓包）
	SubSystemInfo    = 0x07 // 读取系统信息
	SubSetMode       = 0x09 // 设备通道默认工作模式（所有通道）
	SubSetSingleMode = 0x0E // 设置通道的工作方式（单次有效）
	SubSetAllMode    = 0x0F // 设置所用通道的工作方式（单次有效）
)

// 工作模式
const (
	ModeOff = 0x00
	ModeHUB = 0x01
	ModeOTG = 0x02
)

// 应答码
const (
	RespOK            = 0x00
	RespInvalidParent = 0x01
	RespInvalidSub    = 0x02
	RespCRCError      = 0x03
	RespSetFailed     = 0x08
)

// crcTable 文档中的 CRC 字节余式表（CRC-16-CCITT 0x1021 查表法）
var crcTable = [256]uint16{
	0x0000, 0x1021, 0x2042, 0x3063, 0x4084, 0x50a5, 0x60c6, 0x70e7,
	0x8108, 0x9129, 0xa14a, 0xb16b, 0xc18c, 0xd1ad, 0xe1ce, 0xf1ef,
	0x1231, 0x0210, 0x3273, 0x2252, 0x52b5, 0x4294, 0x72f7, 0x62d6,
	0x9339, 0x8318, 0xb37b, 0xa35a, 0xd3bd, 0xc39c, 0xf3ff, 0xe3de,
	0x2462, 0x3443, 0x0420, 0x1401, 0x64e6, 0x74c7, 0x44a4, 0x5485,
	0xa56a, 0xb54b, 0x8528, 0x9509, 0xe5ee, 0xf5cf, 0xc5ac, 0xd58d,
	0x3653, 0x2672, 0x1611, 0x0630, 0x76d7, 0x66f6, 0x5695, 0x46b4,
	0xb75b, 0xa77a, 0x9719, 0x8738, 0xf7df, 0xe7fe, 0xd79d, 0xc7bc,
	0x48c4, 0x58e5, 0x6886, 0x78a7, 0x0840, 0x1861, 0x2802, 0x3823,
	0xc9cc, 0xd9ed, 0xe98e, 0xf9af, 0x8948, 0x9969, 0xa90a, 0xb92b,
	0x5af5, 0x4ad4, 0x7ab7, 0x6a96, 0x1a71, 0x0a50, 0x3a33, 0x2a12,
	0xdbfd, 0xcbdc, 0xfbbf, 0xeb9e, 0x9b79, 0x8b58, 0xbb3b, 0xab1a,
	0x6ca6, 0x7c87, 0x4ce4, 0x5cc5, 0x2c22, 0x3c03, 0x0c60, 0x1c41,
	0xedae, 0xfd8f, 0xcdec, 0xddcd, 0xad2a, 0xbd0b, 0x8d68, 0x9d49,
	0x7e97, 0x6eb6, 0x5ed5, 0x4ef4, 0x3e13, 0x2e32, 0x1e51, 0x0e70,
	0xff9f, 0xefbe, 0xdfdd, 0xcffc, 0xbf1b, 0xaf3a, 0x9f59, 0x8f78,
	0x9188, 0x81a9, 0xb1ca, 0xa1eb, 0xd10c, 0xc12d, 0xf14e, 0xe16f,
	0x1080, 0x00a1, 0x30c2, 0x20e3, 0x5004, 0x4025, 0x7046, 0x6067,
	0x83b9, 0x9398, 0xa3fb, 0xb3da, 0xc33d, 0xd31c, 0xe37f, 0xf35e,
	0x02b1, 0x1290, 0x22f3, 0x32d2, 0x4235, 0x5214, 0x6277, 0x7256,
	0xb5ea, 0xa5cb, 0x95a8, 0x8589, 0xf56e, 0xe54f, 0xd52c, 0xc50d,
	0x34e2, 0x24c3, 0x14a0, 0x0481, 0x7466, 0x6447, 0x5424, 0x4405,
	0xa7db, 0xb7fa, 0x8799, 0x97b8, 0xe75f, 0xf77e, 0xc71d, 0xd73c,
	0x26d3, 0x36f2, 0x0691, 0x16b0, 0x6657, 0x7676, 0x4615, 0x5634,
	0xd94c, 0xc96d, 0xf90e, 0xe92f, 0x99c8, 0x89e9, 0xb98a, 0xa9ab,
	0x5844, 0x4865, 0x7806, 0x6827, 0x18c0, 0x08e1, 0x3882, 0x28a3,
	0xcb7d, 0xdb5c, 0xeb3f, 0xfb1e, 0x8bf9, 0x9bd8, 0xabbb, 0xbb9a,
	0x4a75, 0x5a54, 0x6a37, 0x7a16, 0x0af1, 0x1ad0, 0x2ab3, 0x3a92,
	0xfd2e, 0xed0f, 0xdd6c, 0xcd4d, 0xbdaa, 0xad8b, 0x9de8, 0x8dc9,
	0x7c26, 0x6c07, 0x5c64, 0x4c45, 0x3ca2, 0x2c83, 0x1ce0, 0x0cc1,
	0xef1f, 0xff3e, 0xcf5d, 0xdf7c, 0xaf9b, 0xbfba, 0x8fd9, 0x9ff8,
	0x6e17, 0x7e36, 0x4e55, 0x5e74, 0x2e93, 0x3eb2, 0x0ed1, 0x1ef0,
}

// CRC16 文档算法：CRC-16-CCITT 查表法，initdata=0xFFFF，校验部分为报文头+报文主体
func CRC16(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		high := byte(crc >> 8)
		crc = (crc << 8) ^ crcTable[high^b]
	}
	return crc
}

// BuildPacket 组包：报文头(5) + 报文主体 + CRC16(2)。长度 2 字节与例子程序一致用大端（高字节在前）
func BuildPacket(parentCmd, subCmd byte, body []byte) []byte {
	bodyLen := 1 + len(body) // 子指令 1 字节 + 数据
	header := []byte{
		HeaderByte0, HeaderByte1,
		byte(bodyLen >> 8), byte(bodyLen), // 大端：高字节在前
		parentCmd,
	}
	payload := append(append([]byte{}, header...), append([]byte{subCmd}, body...)...)
	crc := CRC16(payload)
	payload = append(payload, byte(crc), byte(crc>>8))
	return payload
}

// ParsePacket 解析一帧：返回 父指令、子指令、应答码(仅设备设置)、数据、error
func ParsePacket(raw []byte) (parentCmd, subCmd, respCode byte, data []byte, err error) {
	if len(raw) < 5+2 {
		return 0, 0, 0, nil, fmt.Errorf("packet too short")
	}
	if raw[0] != HeaderByte0 || raw[1] != HeaderByte1 {
		return 0, 0, 0, nil, fmt.Errorf("invalid header")
	}
	bodyLen := int(uint16(raw[2]) | uint16(raw[3])<<8) // 小端 (尝试)
	total := 5 + bodyLen + 2
	if len(raw) < total {
		return 0, 0, 0, nil, fmt.Errorf("incomplete packet")
	}
	payload := raw[:5+bodyLen]
	crcGot := uint16(raw[5+bodyLen]) | uint16(raw[5+bodyLen+1])<<8
	if CRC16(payload) != crcGot {
		return 0, 0, 0, nil, fmt.Errorf("crc mismatch")
	}
	parentCmd = raw[4]
	subCmd = raw[5]
	if parentCmd == ParentSet && bodyLen >= 2 {
		respCode = raw[6]
		data = raw[7 : 5+bodyLen]
	} else {
		data = raw[6 : 5+bodyLen]
	}
	return parentCmd, subCmd, respCode, data, nil
}

// ChannelStatus 单通道状态：接入情况 + 工作模式
type ChannelStatus struct {
	PlugState byte   // 0 无主板 1 有主板 2 正在拔出 3 正在接入
	Mode      byte   // 0 关机 1 HUB 2 OTG
	ModeStr   string // "OFF" / "HUB" / "OTG"
}

func ParseChannelStatus(status byte) ChannelStatus {
	plug := status / 16
	mode := status % 16
	var modeStr string
	switch mode {
	case ModeOff:
		modeStr = "OFF"
	case ModeHUB:
		modeStr = "HUB"
	case ModeOTG:
		modeStr = "OTG"
	default:
		modeStr = fmt.Sprintf("0x%02X", mode)
	}
	return ChannelStatus{PlugState: plug, Mode: mode, ModeStr: modeStr}
}

// SystemInfo 0x07 返回中的设备信息
type SystemInfo struct {
	UID           uint32
	Version       float32
	MAC           net.HardwareAddr
	IP            net.IP
	Mask          net.IP
	Gateway       net.IP
	DefaultModes  [20]byte
	FanMode       byte
	ChannelStatus [20]ChannelStatus
}

// ParseSystemInfoBody 解析 0x07 报文数据。文档：UID(4)+版本(4)+MAC(6)+IP(4)+掩码(4)+网关(4)+默认模式20+风扇(1)+当前通道状态(20)
func ParseSystemInfoBody(data []byte) (*SystemInfo, error) {
	// 4+4+6+4+4+4+20+1+20 = 67 bytes
	if len(data) < 67 {
		return nil, fmt.Errorf("system info body too short: %d < 67", len(data))
	}

	var info SystemInfo

	// UID (4 bytes LE)
	info.UID = binary.LittleEndian.Uint32(data[0:4])

	// Version (4 bytes Float LE)
	bits := binary.LittleEndian.Uint32(data[4:8])
	info.Version = math.Float32frombits(bits)

	// MAC (6 bytes)
	info.MAC = make(net.HardwareAddr, 6)
	copy(info.MAC, data[8:14])

	// IP (4 bytes)
	info.IP = make(net.IP, 4)
	copy(info.IP, data[14:18])

	// Mask (4 bytes)
	info.Mask = make(net.IP, 4)
	copy(info.Mask, data[18:22])

	// Gateway (4 bytes)
	info.Gateway = make(net.IP, 4)
	copy(info.Gateway, data[22:26])

	// Default Modes (20 bytes)
	copy(info.DefaultModes[:], data[26:46])

	// Fan Mode (1 byte)
	info.FanMode = data[46]

	// Channel Status (20 bytes)
	statusData := data[47:67]
	for i := 0; i < 20; i++ {
		info.ChannelStatus[i] = ParseChannelStatus(statusData[i])
	}
	return &info, nil
}

// BuildConnectPacket 建立连接 0x02
func BuildConnectPacket() []byte {
	return BuildPacket(ParentSet, SubConnect, nil)
}

// BuildSystemInfoPacket 读取系统信息 0x07
func BuildSystemInfoPacket() []byte {
	return BuildPacket(ParentSet, SubSystemInfo, nil)
}

// BuildSetAllModePacket 设置所有通道工作模式 0x0F（单次有效）
// 附加数据包为全 0x00
func BuildSetAllModePacket(mode byte) []byte {
	body := make([]byte, 1+20)
	body[0] = mode
	// 后面 20 字节默认 0x00，无需额外赋值
	return BuildPacket(ParentSet, SubSetAllMode, body)
}

// BuildSetSingleModePacket 设置单通道工作模式 0x0E（单次有效）
// 通道号(1 byte)+工作模式(1 byte)
func BuildSetSingleModePacket(channelIdx byte, mode byte) []byte {
	return BuildPacket(ParentSet, SubSetSingleMode, []byte{channelIdx, mode})
}

// BuildRestartSinglePacket 重启单路通道 0x10
func BuildRestartSinglePacket(channelIdx byte) []byte {
	return BuildPacket(ParentSet, 0x10, []byte{channelIdx})
}

// BuildRestartAllPacket 重启所有通道 0x11
// 附加数据包为全 0x00
func BuildRestartAllPacket() []byte {
	body := make([]byte, 20)
	// 默认 0x00
	return BuildPacket(ParentSet, 0x11, body)
}

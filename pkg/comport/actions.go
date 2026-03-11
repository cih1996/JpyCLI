package comport

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const ReadTimeout = 3 * time.Second

// ListPortsOnOpenError 在打开失败时列出可用端口并返回带提示的 error
func ListPortsOnOpenError(portName string, openErr error) error {
	ports, err := ListPorts()
	fmt.Fprintf(os.Stderr, "打开 %s 失败: %v\n", portName, openErr)
	if err != nil {
		return fmt.Errorf("%w（可用端口列举失败: %v）", openErr, err)
	}
	candidates := FilterCandidatePorts(ports)
	if len(candidates) == 0 {
		fmt.Fprintf(os.Stderr, "当前无可用串口（或仅有蓝牙/调试口），请检查设备是否插好、驱动是否安装。\n")
		return openErr
	}
	fmt.Fprintf(os.Stderr, "当前可用串口（请用 -port 指定其一，名称要完整，如末尾数字不要少写）:\n")
	for _, p := range candidates {
		fmt.Fprintf(os.Stderr, "  -port %s\n", p)
	}
	if len(candidates) == 1 {
		fmt.Fprintf(os.Stderr, "建议直接: -port %s\n", candidates[0])
	}
	return openErr
}

// FilterCandidatePorts 过滤出较可能是 USB 串口（CH340 等）的端口，用于自动选择或提示
func FilterCandidatePorts(ports []string) []string {
	var out []string
	for _, p := range ports {
		base := p
		if i := strings.LastIndex(p, "/"); i >= 0 {
			base = p[i+1:]
		}
		// 排除蓝牙、调试控制台
		if strings.Contains(base, "Bluetooth") || strings.Contains(base, "debug-console") {
			continue
		}
		// 保留 cu.*（Mac/Linux）或 COM*（Windows）
		if strings.HasPrefix(base, "cu.") || strings.HasPrefix(base, "COM") {
			out = append(out, p)
		}
	}
	return out
}

// ResolvePort 若 port 为空，则尝试自动选用唯一候选端口；同一 CH340 可能对应 cu.wch 与 cu.usbserial 两个节点，优先用 wch
func ResolvePort(port string, subcmd string) (string, error) {
	if port != "" {
		return port, nil
	}
	ports, err := ListPorts()
	if err != nil {
		return "", fmt.Errorf("列举端口失败: %w", err)
	}
	candidates := FilterCandidatePorts(ports)
	if len(candidates) == 0 {
		return "", fmt.Errorf("未找到可用串口，请插好设备后执行 com-cli list 查看")
	}
	if len(candidates) == 1 {
		return candidates[0], nil
	}
	// 若有两个候选且成对（同一设备：wch + usbserial 同编号），优先用 wch 以便 "go run . devices" 可直接跑
	if len(candidates) == 2 {
		var wch, other string
		for _, p := range candidates {
			if strings.Contains(p, "wch") {
				wch = p
			} else {
				other = p
			}
		}
		if wch != "" && other != "" {
			return wch, nil
		}
	}
	fmt.Fprintf(os.Stderr, "%s: 找到多个端口，请指定 -port:\n", subcmd)
	for _, p := range candidates {
		fmt.Fprintf(os.Stderr, "  %s\n", p)
	}
	return "", fmt.Errorf("请指定 -port 参数")
}

func RunDevices(portName string, debug bool, skipConnect bool) (*DeviceListResult, error) {
	port, err := OpenPort(portName, DefaultPortConfig())
	if err != nil {
		ListPortsOnOpenError(portName, err)
		return nil, err
	}
	defer port.Close()

	DrainBuffer(port)

	// 1. 建立连接 0x02 (原 0x04)
	if !skipConnect {
		connectPkt := BuildConnectPacket()
		if err := SendPacket(port, connectPkt); err != nil {
			return nil, err
		}
		time.Sleep(50 * time.Millisecond) // 给设备处理时间

		// 使用循环读取，过滤掉可能存在的干扰包
		_, data, err := ReadExpectedPacket(port, ReadTimeout, ParentSet, SubConnect)
		if err != nil {
			return nil, fmt.Errorf("建连应答: %w（若设备无反应，请检查波特率是否 115200、接线与设备上电）", err)
		}

		// 校验建连结果
		if len(data) > 0 && data[0] != 0x00 {
			// 非成功状态，尝试继续
		}
	}

	sysPkt := BuildSystemInfoPacket()
	if err := SendPacket(port, sysPkt); err != nil {
		return nil, err
	}

	// 读取系统信息响应，同样需要过滤通知包
	raw, data, err := ReadExpectedPacket(port, ReadTimeout, ParentSet, SubSystemInfo)
	if err != nil {
		return nil, fmt.Errorf("读系统信息: %w", err)
	}

	// 这里不需要再 ParsePacket 了，ReadExpectedPacket 已经校验过 parent/sub
	// 我们直接校验 resp
	_, _, resp, _, _ := ParsePacket(raw)
	if resp != RespOK {
		// 尝试解析错误详情
		status := byte(0xFF)
		if len(data) > 0 {
			status = data[0]
		}
		return nil, fmt.Errorf("设备应答错误: %02X, 状态码: %s", resp, GetStatusMsg(status))
	}

	info, err := ParseSystemInfoBody(data)
	if err != nil {
		return nil, fmt.Errorf("解析系统信息: %w", err)
	}

	// 输出 JSON
	channels := make([]DeviceChannel, 0, 20)
	plugStr := func(b byte) string {
		switch b {
		case 0:
			return "无主板"
		case 1:
			return "有主板"
		case 2:
			return "正在拔出"
		case 3:
			return "正在接入"
		default:
			return fmt.Sprintf("0x%02X", b)
		}
	}
	for i := 0; i < 20; i++ {
		c := info.ChannelStatus[i]
		channels = append(channels, DeviceChannel{
			Channel:  i + 1,
			Plug:     plugStr(c.PlugState),
			Mode:     c.ModeStr,
			ModeCode: c.Mode,
		})
	}
	result := DeviceListResult{
		Port:     portName,
		UID:      fmt.Sprintf("%d", info.UID),
		Version:  fmt.Sprintf("%.2f", info.Version),
		MAC:      info.MAC.String(),
		IP:       info.IP.String(),
		Mask:     info.Mask.String(),
		Gateway:  info.Gateway.String(),
		FanMode:  fmt.Sprintf("0x%02X", info.FanMode),
		Channels: channels,
	}
	return &result, nil
}

func RunSetMode(portName, modeStr string, channel int, debug, skipConnect bool) error {
	res, err := RunSetModeBatch(portName, []SetModeItem{{Mode: modeStr, Channel: channel}}, debug, skipConnect)
	// 如果有错误返回，说明是打开串口等严重错误
	if err != nil {
		return err
	}
	// 否则检查单个结果
	if len(res) > 0 && !res[0].Success {
		return fmt.Errorf(res[0].Error)
	}
	return nil
}

type SetModeItem struct {
	Mode    string
	Channel int
}

type SetModeResult struct {
	Channel int
	Success bool
	Error   string
}

// RunSetModeBatch 批量设置模式（复用同一连接）
func RunSetModeBatch(portName string, items []SetModeItem, debug, skipConnect bool) ([]SetModeResult, error) {
	results := make([]SetModeResult, len(items))
	for i, item := range items {
		results[i] = SetModeResult{Channel: item.Channel, Success: false, Error: "未执行"}
	}

	port, err := OpenPort(portName, DefaultPortConfig())
	if err != nil {
		ListPortsOnOpenError(portName, err)
		for i := range results {
			results[i].Error = fmt.Sprintf("打开串口失败: %v", err)
		}
		return results, err
	}
	defer port.Close()

	DrainBuffer(port)

	if !skipConnect {
		connectPkt := BuildConnectPacket()
		if err := SendPacket(port, connectPkt); err != nil {
			for i := range results {
				results[i].Error = fmt.Sprintf("发送建连指令失败: %v", err)
			}
			return results, err
		}
		_, data, err := ReadExpectedPacket(port, ReadTimeout, ParentSet, SubConnect)
		if err != nil {
			for i := range results {
				results[i].Error = fmt.Sprintf("建连应答失败: %v", err)
			}
			return results, fmt.Errorf("建连应答: %w", err)
		}
		_ = data
	}

	for i, item := range items {
		var mode byte
		switch item.Mode {
		case "hub", "HUB":
			mode = ModeHUB
		case "otg", "OTG":
			mode = ModeOTG
		default:
			results[i].Error = fmt.Sprintf("mode 必须是 hub 或 otg，当前: %s", item.Mode)
			continue
		}

		var setPkt []byte
		var expectSub byte

		if item.Channel == 0 {
			setPkt = BuildSetAllModePacket(mode)
			expectSub = SubSetAllMode
		} else {
			if item.Channel < 1 || item.Channel > 20 {
				results[i].Error = "通道号必须在 1-20 之间"
				continue
			}
			setPkt = BuildSetSingleModePacket(byte(item.Channel-1), mode)
			expectSub = SubSetSingleMode
		}

		if err := SendPacket(port, setPkt); err != nil {
			results[i].Error = fmt.Sprintf("发送指令失败: %v", err)
			continue
		}

		raw, data, err := ReadExpectedPacket(port, ReadTimeout, ParentSet, expectSub)
		if err != nil {
			results[i].Error = fmt.Sprintf("读应答失败: %v", err)
			continue
		}

		_, _, resp, _, err := ParsePacket(raw)
		if err != nil {
			results[i].Error = fmt.Sprintf("解析应答失败: %v", err)
			continue
		}
		if resp != RespOK {
			status := byte(0xFF)
			if len(data) > 0 {
				status = data[0]
			}
			results[i].Error = fmt.Sprintf("设备应答错误: %02X, 状态码: %s", resp, GetStatusMsg(status))
			continue
		}

		resCode := byte(0xFF)
		if len(data) >= 1 {
			resCode = data[0]
		}

		if resCode != 0x00 && resCode != 0x02 && resCode != 0x03 {
			results[i].Error = fmt.Sprintf("设置失败，结果: %s", GetStatusMsg(resCode))
			continue
		}

		results[i].Success = true
		results[i].Error = ""

		// 稍微延时，避免发包过快
		time.Sleep(50 * time.Millisecond)
	}

	return results, nil
}

func RunRestart(portName string, channel int, debug, skipConnect bool) error {
	port, err := OpenPort(portName, DefaultPortConfig())
	if err != nil {
		ListPortsOnOpenError(portName, err)
		return err
	}
	defer port.Close()

	DrainBuffer(port)

	if !skipConnect {
		connectPkt := BuildConnectPacket()
		if err := SendPacket(port, connectPkt); err != nil {
			return err
		}
		_, data, err := ReadExpectedPacket(port, ReadTimeout, ParentSet, SubConnect)
		if err != nil {
			return fmt.Errorf("建连应答: %w", err)
		}
		_ = data
	}

	var pkt []byte
	var expectSub byte

	if channel == 0 {
		pkt = BuildRestartAllPacket()
		expectSub = 0x11
	} else {
		if channel < 1 || channel > 20 {
			return fmt.Errorf("通道号必须在 1-20 之间")
		}
		pkt = BuildRestartSinglePacket(byte(channel - 1)) // 0-19
		expectSub = 0x10
	}

	if err := SendPacket(port, pkt); err != nil {
		return err
	}

	raw, data, err := ReadExpectedPacket(port, ReadTimeout, ParentSet, expectSub)
	if err != nil {
		return fmt.Errorf("读重启应答: %w", err)
	}

	_, _, resp, _, err := ParsePacket(raw)
	if err != nil {
		return fmt.Errorf("解析应答: %w", err)
	}
	if resp != RespOK {
		status := byte(0xFF)
		if len(data) > 0 {
			status = data[0]
		}
		return fmt.Errorf("重启失败，应答码: %02X, 状态码: %s", resp, GetStatusMsg(status))
	}

	// 0x10/0x11 应答第一字节都是结果
	resCode := byte(0xFF)
	if len(data) >= 1 {
		resCode = data[0]
	}

	if resCode != 0x00 && resCode != 0x03 { // 0x03=所有通道已关机，也算成功
		return fmt.Errorf("重启失败，结果: %s", GetStatusMsg(resCode))
	}

	return nil
}

type RestartItem struct {
	Channel int
}

type RestartResult struct {
	Channel int
	Success bool
	Error   string
}

// RunRestartBatch 批量重启（复用同一连接）
func RunRestartBatch(portName string, items []RestartItem, debug, skipConnect bool) ([]RestartResult, error) {
	results := make([]RestartResult, len(items))
	for i, item := range items {
		results[i] = RestartResult{Channel: item.Channel, Success: false, Error: "未执行"}
	}

	port, err := OpenPort(portName, DefaultPortConfig())
	if err != nil {
		ListPortsOnOpenError(portName, err)
		for i := range results {
			results[i].Error = fmt.Sprintf("打开串口失败: %v", err)
		}
		return results, err
	}
	defer port.Close()

	DrainBuffer(port)

	if !skipConnect {
		connectPkt := BuildConnectPacket()
		if err := SendPacket(port, connectPkt); err != nil {
			for i := range results {
				results[i].Error = fmt.Sprintf("发送建连指令失败: %v", err)
			}
			return results, err
		}
		_, data, err := ReadExpectedPacket(port, ReadTimeout, ParentSet, SubConnect)
		if err != nil {
			for i := range results {
				results[i].Error = fmt.Sprintf("建连应答失败: %v", err)
			}
			return results, fmt.Errorf("建连应答: %w", err)
		}
		_ = data
	}

	for i, item := range items {
		var pkt []byte
		var expectSub byte

		if item.Channel == 0 {
			pkt = BuildRestartAllPacket()
			expectSub = 0x11
		} else {
			if item.Channel < 1 || item.Channel > 20 {
				results[i].Error = "通道号必须在 1-20 之间"
				continue
			}
			pkt = BuildRestartSinglePacket(byte(item.Channel - 1)) // 0-19
			expectSub = 0x10
		}

		if err := SendPacket(port, pkt); err != nil {
			results[i].Error = fmt.Sprintf("发送指令失败: %v", err)
			continue
		}

		raw, data, err := ReadExpectedPacket(port, ReadTimeout, ParentSet, expectSub)
		if err != nil {
			results[i].Error = fmt.Sprintf("读应答失败: %v", err)
			continue
		}

		_, _, resp, _, err := ParsePacket(raw)
		if err != nil {
			results[i].Error = fmt.Sprintf("解析应答失败: %v", err)
			continue
		}
		if resp != RespOK {
			status := byte(0xFF)
			if len(data) > 0 {
				status = data[0]
			}
			results[i].Error = fmt.Sprintf("重启失败，应答码: %02X, 状态码: %s", resp, GetStatusMsg(status))
			continue
		}

		resCode := byte(0xFF)
		if len(data) >= 1 {
			resCode = data[0]
		}

		if resCode != 0x00 && resCode != 0x03 {
			results[i].Error = fmt.Sprintf("重启失败，结果: %s", GetStatusMsg(resCode))
			continue
		}

		results[i].Success = true
		results[i].Error = ""

		// 稍微延时
		time.Sleep(50 * time.Millisecond)
	}

	return results, nil
}

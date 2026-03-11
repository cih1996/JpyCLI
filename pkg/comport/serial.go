package comport

import (
	"fmt"
	"io"
	"runtime"
	"strings"
	"time"

	"go.bug.st/serial"
)

// ListPorts 列举所有可用 COM 口
func ListPorts() ([]string, error) {
	list, err := serial.GetPortsList()
	if err != nil {
		return nil, fmt.Errorf("列举串口失败: %w", err)
	}
	if len(list) == 0 {
		return []string{}, nil
	}
	return list, nil
}

// PortConfig 串口配置（与协议文档一致：常见 115200 8N1）
type PortConfig struct {
	BaudRate int
	DataBits int
	StopBits int
	Parity   serial.Parity
}

// DefaultPortConfig 默认：115200 8N1
func DefaultPortConfig() PortConfig {
	return PortConfig{
		BaudRate: 115200,
		DataBits: 8,
		StopBits: 1,
		Parity:   serial.NoParity,
	}
}

// SerialPort 包装 serial.Port，统一返回接口
type SerialPort struct {
	serial.Port
}

// OpenPort 打开指定串口
func OpenPort(portName string, cfg PortConfig) (io.ReadWriteCloser, error) {
	stopBits := serial.OneStopBit
	if cfg.StopBits == 2 {
		stopBits = serial.TwoStopBits
	}

	// 非 Windows 平台：初始 DTR/RTS 设为 false
	var initialBits *serial.ModemOutputBits
	if runtime.GOOS != "windows" {
		initialBits = &serial.ModemOutputBits{DTR: false, RTS: false}
	}

	p, err := serial.Open(portName, &serial.Mode{
		BaudRate:          cfg.BaudRate,
		DataBits:          cfg.DataBits,
		Parity:            cfg.Parity,
		StopBits:          stopBits,
		InitialStatusBits: initialBits,
	})
	if err != nil {
		return nil, fmt.Errorf("打开串口 %s 失败: %w", portName, err)
	}

	// 设置读超时 100ms，避免 Read 无限阻塞
	_ = p.SetReadTimeout(100 * time.Millisecond)

	// 清空缓冲区
	_ = p.ResetInputBuffer()
	_ = p.ResetOutputBuffer()

	if runtime.GOOS != "windows" {
		time.Sleep(300 * time.Millisecond)
	}

	return &SerialPort{Port: p}, nil
}

func IsClosedErr(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "bad file descriptor") ||
		strings.Contains(s, "closed") ||
		strings.Contains(s, "use of closed") ||
		strings.Contains(s, "Port has been closed")
}

// SetReadTimeout 设置读超时
func SetReadTimeout(port io.Reader, t time.Duration) error {
	if sp, ok := port.(*SerialPort); ok {
		return sp.Port.SetReadTimeout(t)
	}
	return nil
}

// ReadPacket 从串口读一帧
func ReadPacket(port io.Reader, timeout time.Duration) ([]byte, error) {
	maxScanBytes := 4096
	scannedBytes := 0

	header := make([]byte, 5)

	// 1. 寻找 0xDC
	deadline := time.Now().Add(timeout)
	for {
		if scannedBytes >= maxScanBytes {
			return nil, fmt.Errorf("在 %d 字节内未找到报文头 0xDC", maxScanBytes)
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("read timeout during scan")
		}

		buf1 := make([]byte, 1)
		if _, err := readFullUntil(port, buf1, deadline); err != nil {
			return nil, fmt.Errorf("扫描报文头失败: %w", err)
		}
		scannedBytes++

		if buf1[0] == HeaderByte0 {
			buf2 := make([]byte, 1)
			if _, err := readFullUntil(port, buf2, deadline); err != nil {
				return nil, fmt.Errorf("扫描报文头第二字节失败: %w", err)
			}
			scannedBytes++

			if buf2[0] == HeaderByte1 {
				header[0] = HeaderByte0
				header[1] = HeaderByte1
				if _, err := readFullUntil(port, header[2:], deadline); err != nil {
					return nil, fmt.Errorf("读报文头剩余部分失败: %w", err)
				}
				break
			}
		}
	}

	// 解析长度 (小端序)
	bodyLen := int(uint16(header[2]) | uint16(header[3])<<8)
	restLen := bodyLen + 2 // body + CRC(2)
	rest := make([]byte, restLen)

	if _, err := readFullUntil(port, rest, deadline); err != nil {
		return nil, fmt.Errorf("读报文主体失败: %w", err)
	}

	full := append(header, rest...)
	return full, nil
}

// readFullUntil 读取指定长度，直到填满 buf、出错或超过 deadline
func readFullUntil(r io.Reader, buf []byte, deadline time.Time) (int, error) {
	total := 0
	for total < len(buf) {
		if time.Now().After(deadline) {
			return total, fmt.Errorf("read timeout")
		}

		n, err := r.Read(buf[total:])
		if n > 0 {
			total += n
		}
		if err != nil {
			return total, err
		}
		if n == 0 {
			continue
		}
	}
	return total, nil
}

// SendPacket 发送一帧
func SendPacket(port io.Writer, packet []byte) error {
	_, err := port.Write(packet)
	if err != nil {
		return fmt.Errorf("发送报文失败: %w", err)
	}
	return nil
}

package comport

import (
	"fmt"
	"io"
	"os"
	"time"
)

// PacketStatusMap 响应状态码说明
var PacketStatusMap = map[byte]string{
	0x00: "操作正常",
	0x01: "无效父指令",
	0x02: "无效子指令",
	0x03: "校验错误",
	0x06: "需要先进入串口连接模式",
	0x07: "需要先进入网络连接模式",
	0x08: "设置失败",
	0x09: "固件更新失败",
}

func GetStatusMsg(code byte) string {
	if msg, ok := PacketStatusMap[code]; ok {
		return fmt.Sprintf("%02X (%s)", code, msg)
	}
	return fmt.Sprintf("%02X", code)
}

// DrainBuffer 清空串口读取缓冲区
func DrainBuffer(port io.Reader) {
	buf := make([]byte, 1024)
	// 尝试设置极短超时
	// _ = SetReadTimeout(port, 100*time.Millisecond)
	for {
		n, err := port.Read(buf)
		if n > 0 {
			fmt.Fprintf(os.Stderr, "[Drain] 丢弃缓冲区残留 %d 字节\n", n)
		}
		if err != nil || n == 0 {
			break
		}
	}
}

// ReadExpectedPacket 读取数据包直到匹配期望的 parent/sub 或超时
func ReadExpectedPacket(port io.Reader, timeout time.Duration, expectParent byte, expectSub byte) ([]byte, []byte, error) {
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			return nil, nil, fmt.Errorf("等待指令 %02X-%02X 超时", expectParent, expectSub)
		}

		// 计算剩余时间
		timeLeft := time.Until(deadline)
		if timeLeft < 100*time.Millisecond {
			timeLeft = 100 * time.Millisecond
		}

		raw, err := ReadPacket(port, timeLeft)
		if err != nil {
			// 如果在 deadline 之前超时，说明只是暂无数据，继续重试
			if time.Now().Before(deadline) {
				// 短暂休眠避免死循环空转 (虽然 ReadPacket 内部有 100ms 超时)
				time.Sleep(10 * time.Millisecond)
				continue
			}
			return nil, nil, err
		}

		// 再次检查超时，防止 ReadPacket 耗时过长或在洪水攻击下死循环
		if time.Now().After(deadline) {
			return nil, nil, fmt.Errorf("等待指令 %02X-%02X 超时", expectParent, expectSub)
		}

		parent, sub, _, data, err := ParsePacket(raw)
		if err != nil {
			fmt.Fprintf(os.Stderr, "解析数据包失败: %v (忽略)\n", err)
			continue
		}

		if parent == expectParent && sub == expectSub {
			return raw, data, nil
		}

		// 如果是通知包 (Parent=0x00)，打印日志并继续等待
		if parent == 0x00 {
			// fmt.Fprintf(os.Stderr, "收到通知包 (Sub=%02X), 继续等待目标包...\n", sub)
			continue
		}

		fmt.Fprintf(os.Stderr, "收到非目标包 (Parent=%02X Sub=%02X), 继续等待...\n", parent, sub)
	}
}

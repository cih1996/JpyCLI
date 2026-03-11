package flash

import (
	"fmt"
	"os"
	"time"
)

// 日志级别
const (
	levelInfo  = "INFO"
	levelWarn  = "WARN"
	levelError = "ERROR"
)

// logWithPrefix 带前缀的日志输出
// 格式: [时间戳] [COM口-通道] [级别] 消息
func logWithPrefix(com string, channel int, level string, format string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	msg := fmt.Sprintf(format, args...)

	var prefix string
	if com != "" && channel > 0 {
		prefix = fmt.Sprintf("[%s] [%s-CH%02d] [%s]", timestamp, com, channel, level)
	} else if com != "" {
		prefix = fmt.Sprintf("[%s] [%s] [%s]", timestamp, com, level)
	} else {
		prefix = fmt.Sprintf("[%s] [%s]", timestamp, level)
	}

	fmt.Fprintf(os.Stderr, "%s %s\n", prefix, msg)
}

// logInfo 信息日志
func logInfo(com string, channel int, format string, args ...interface{}) {
	logWithPrefix(com, channel, levelInfo, format, args...)
}

// logWarn 警告日志
func logWarn(com string, channel int, format string, args ...interface{}) {
	logWithPrefix(com, channel, levelWarn, format, args...)
}

// logError 错误日志
func logError(com string, channel int, format string, args ...interface{}) {
	logWithPrefix(com, channel, levelError, format, args...)
}

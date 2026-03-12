package main

import (
	// 必须先导入 logs 并设置配置，再导入其他包
	// 这样可以在 adminApi 包初始化之前抑制日志
	"cnb.cool/accbot/goTool/logs"
	_ "jpy-cli/internal/cmd/stress/silentinit" // 静默初始化
	"jpy-cli/internal/cmd"
)

func init() {
	// 双重保险：再次设置日志配置
	silentLogConfig := `
level: error
console: false
file: false
`
	_ = logs.SetLoggerConfig(silentLogConfig, 0)
}

func main() {
	cmd.Execute()
}

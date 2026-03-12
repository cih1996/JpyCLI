// Package silentinit 用于在其他包之前初始化日志配置
// 必须在 main.go 中最先导入此包
package silentinit

import "cnb.cool/accbot/goTool/logs"

func init() {
	// 在所有其他包初始化之前抑制 SDK 内部日志
	silentLogConfig := `
level: error
console: false
file: false
`
	_ = logs.SetLoggerConfig(silentLogConfig, 0)
}

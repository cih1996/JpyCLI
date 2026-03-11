package cmd

import (
	"fmt"
	"jpy-cli/internal/cmd/com"
	"jpy-cli/internal/cmd/device"
	"jpy-cli/internal/cmd/file"
	"jpy-cli/internal/cmd/flash"
	"jpy-cli/pkg/logger"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	debug    bool
	logLevel string
)

var rootCmd = &cobra.Command{
	Use:   "jpy",
	Short: "JPY 中间件命令行工具（AI 模式）",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// 日志初始化（写文件，不依赖配置）
		home, _ := os.UserHomeDir()
		logDir := filepath.Join(home, ".jpy", "logs")
		_ = os.MkdirAll(logDir, 0755)
		logPath := filepath.Join(logDir, "jpy.log")

		level := "info"
		enableConsole := false
		if debug {
			level = "debug"
			enableConsole = true
		} else if logLevel != "" {
			level = logLevel
		}

		_ = logger.Init(logger.Options{
			Level:    level,
			FilePath: logPath,
			Console:  enableConsole,
			File:     true,
		})
	},
}

func Execute() {
	// --remote 拦截：在 Cobra 解析之前检查，命中则转发到远端
	if addr, args := extractRemoteFlag(os.Args[1:]); addr != "" {
		// shell 子命令走专用远程处理
		if len(args) > 0 && args[0] == "shell" {
			remoteShellExec(addr, args[1:])
			return
		}
		remoteExec(addr, args)
		return
	}

	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "启用调试日志")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "日志级别 (debug/info/warn/error)")

	// 核心命令
	rootCmd.AddCommand(device.NewDeviceCmd())
	rootCmd.AddCommand(com.NewComCmd())
	rootCmd.AddCommand(flash.NewFlashCmd())
	rootCmd.AddCommand(file.NewFileCmd())
	rootCmd.AddCommand(newServerCmd())
	rootCmd.AddCommand(newShellCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

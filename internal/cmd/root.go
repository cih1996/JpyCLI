package cmd

import (
	"fmt"
	adminMiddleware "jpy-cli/internal/cmd/admin/middleware"
	cloudCmd "jpy-cli/internal/cmd/cloud"
	configCmd "jpy-cli/internal/cmd/config"
	logCmd "jpy-cli/internal/cmd/log"
	"jpy-cli/internal/cmd/middleware"
	"jpy-cli/internal/cmd/proxy"
	"jpy-cli/internal/cmd/server"
	"jpy-cli/internal/cmd/test"
	"jpy-cli/internal/cmd/tools"
	"jpy-cli/pkg/config"
	"jpy-cli/pkg/logger"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	debug    bool
	logLevel string
)

func loadConfig() *config.Settings {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	configDir := filepath.Join(home, ".jpy")
	configFile := filepath.Join(configDir, "config.yaml")

	// Check if config exists, if not create it
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		_ = os.MkdirAll(configDir, 0755)
		defaultConfig := []byte(`log_level: info
log_output: file # console, file, both
max_concurrency: 5
connect_timeout: 3 # seconds
`)
		if err := os.WriteFile(configFile, defaultConfig, 0644); err != nil {
			fmt.Printf("警告: 创建默认配置文件失败 %s: %v\n", configFile, err)
		} else {
			fmt.Printf("已创建默认配置文件 %s\n", configFile)
		}
	}

	// Read config
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil
	}

	var cfg config.Settings
	if err := yaml.Unmarshal(data, &cfg); err == nil {
		return &cfg
	}
	return nil
}

var rootCmd = &cobra.Command{
	Use:   "jpy",
	Short: "JPY 中间件命令行工具",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Strict log file path: ~/.jpy/logs/jpy.log
		home, err := os.UserHomeDir()
		if err != nil {
			// Critical error: cannot determine home directory
			panic("无法获取用户主目录用于日志: " + err.Error())
		}

		logDir := filepath.Join(home, ".jpy", "logs")
		if err := os.MkdirAll(logDir, 0755); err != nil {
			fmt.Printf("警告: 创建日志目录失败 %s: %v\n", logDir, err)
		}
		logPath := filepath.Join(logDir, "jpy.log")

		level := "info"
		logOutput := "file" // Default to file only

		// Load config from file
		if cfg := loadConfig(); cfg != nil {
			config.GlobalSettings = *cfg
			if cfg.LogLevel != "" {
				level = cfg.LogLevel
			}
			if cfg.LogOutput != "" {
				logOutput = cfg.LogOutput
			}
		}

		// Set default concurrency if not set
		if config.GlobalSettings.MaxConcurrency == 0 {
			config.GlobalSettings.MaxConcurrency = 5
		}

		// Set default connect timeout if not set
		if config.GlobalSettings.ConnectTimeout == 0 {
			config.GlobalSettings.ConnectTimeout = 3
		}

		// Flag overrides
		if debug {
			level = "debug"
			// If user explicitly asks for debug flag, ensure console is on unless configured otherwise?
			// Usually --debug implies console output in many CLIs, but let's stick to strict config + flag logic.
			// For backward compatibility/dev convenience, if debug flag is set, force enable console?
			// User asked for control. Let's make flag ONLY set level, and maybe force console if no config?
			// Let's keep it simple: flag sets level. output is controlled by config or default (file).
			// Wait, previous logic was: debug flag -> console + file.
			// User wants control.
			// Let's say: if debug flag is on, we default to "both" if config didn't specify output.
			if logOutput == "file" {
				logOutput = "both"
			}
		} else if logLevel != "" && logLevel != "info" {
			level = logLevel
		}

		enableConsole := logOutput == "console" || logOutput == "both"
		enableFile := logOutput == "file" || logOutput == "both"

		if enableConsole {
			fmt.Printf("正在初始化日志 level: %s, output: %s\n", level, logOutput)
			if enableFile {
				fmt.Printf("日志文件路径: %s\n", logPath)
			}
		}

		if err := logger.Init(logger.Options{
			Level:    level,
			FilePath: logPath,
			Console:  enableConsole,
			File:     enableFile,
		}); err != nil {
			fmt.Println("警告: 初始化日志失败:", err)
		}
	},
}

func Execute() {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "启用调试日志")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "设置日志级别 (debug, info, warn, error)")
	// SSH server command
	rootCmd.AddCommand(server.NewSSHServerCmd())
	// Web server command
	rootCmd.AddCommand(server.NewWebServerCmd())

	// Middleware commands (局域网中间件)
	rootCmd.AddCommand(middleware.NewMiddlewareCmd())

	// Cloud commands (集控平台远程 API)
	rootCmd.AddCommand(cloudCmd.NewCloudCmd())

	// Tools commands
	rootCmd.AddCommand(tools.NewToolsCmd())

	// Test commands
	rootCmd.AddCommand(test.NewTestCmd())

	// Server Commands
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "启动服务端功能 (SSH/Web)",
	}
	serverCmd.AddCommand(server.NewSSHServerCmd())
	serverCmd.AddCommand(server.NewWebServerCmd())
	rootCmd.AddCommand(serverCmd)

	// Config commands
	rootCmd.AddCommand(configCmd.NewConfigCmd())

	// Proxy command
	rootCmd.AddCommand(proxy.NewProxyCmd())

	// Log commands
	rootCmd.AddCommand(logCmd.NewLogCmd())

	// Admin commands
	adminCmd := &cobra.Command{
		Use:   "admin",
		Short: "管理相关命令",
	}
	middlewareCmd := &cobra.Command{
		Use:   "middleware",
		Short: "中间件服务器管理命令",
	}
	middlewareCmd.AddCommand(adminMiddleware.NewGenerateCmd())
	middlewareCmd.AddCommand(adminMiddleware.NewListCmd())
	middlewareCmd.AddCommand(adminMiddleware.NewGetRootPasswordCmd())
	adminCmd.AddCommand(middlewareCmd)
	rootCmd.AddCommand(adminCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

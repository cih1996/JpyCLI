package cloud

import (
	"encoding/json"
	"fmt"
	"os"

	cloudPkg "jpy-cli/pkg/cloud"

	"github.com/spf13/cobra"
)

// NewConfigCmd 创建 cloud config 命令
func NewConfigCmd() *cobra.Command {
	var (
		serverURL string
		secretKey string
		output    string
	)

	cmd := &cobra.Command{
		Use:   "config",
		Short: "查看或修改集控平台配置",
		Long: `查看或修改集控平台连接配置。

示例:
  jpy cloud config                                   # 查看当前配置
  jpy cloud config --server-url wss://xxx/ws          # 设置服务器地址
  jpy cloud config --secret-key <key>                 # 设置认证密钥
  jpy cloud config --json                             # JSON 格式输出`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := cloudPkg.LoadCloudConfig()
			if err != nil {
				return err
			}

			changed := false

			if cmd.Flags().Changed("server-url") {
				cfg.ServerURL = serverURL
				changed = true
			}

			if cmd.Flags().Changed("secret-key") {
				cfg.Auth.SecretKey = secretKey
				changed = true
			}

			if changed {
				if err := cloudPkg.SaveCloudConfig(cfg); err != nil {
					return err
				}
				fmt.Println("配置已保存。")
			}

			// 输出当前配置
			if output == "json" {
				displayCfg := map[string]interface{}{
					"server_url": cfg.ServerURL,
					"secret_key": maskSecret(cfg.Auth.SecretKey),
				}
				data, _ := json.MarshalIndent(displayCfg, "", "  ")
				fmt.Println(string(data))
			} else {
				fmt.Printf("服务器地址: %s\n", cfg.ServerURL)
				fmt.Printf("认证密钥:   %s\n", maskSecret(cfg.Auth.SecretKey))
				fmt.Printf("配置路径:   %s\n", cloudPkg.GetCloudConfigPath())
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&serverURL, "server-url", "", "集控平台 WebSocket 地址")
	cmd.Flags().StringVar(&secretKey, "secret-key", "", "认证密钥")
	cmd.Flags().StringVarP(&output, "output", "o", "plain", "输出格式 (plain/json)")

	// 初始化改机配置目录子命令
	cmd.AddCommand(newInitConfigsCmd())

	return cmd
}

func maskSecret(s string) string {
	if s == "" {
		return "(未设置)"
	}
	if len(s) <= 10 {
		return "***"
	}
	return s[:5] + "..." + s[len(s)-5:]
}

// newInitConfigsCmd 创建示例改机配置文件
func newInitConfigsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init-configs",
		Short: "创建示例改机配置文件",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := cloudPkg.GetChangeOsConfigsDir()
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("创建目录失败: %w", err)
			}

			configs := map[string]interface{}{
				"zh.json": map[string]string{
					"bs": "wifi", "category": "191", "version": "191",
					"country": "cn", "language": "zh-Hans-CN", "timezone": "Asia/Shanghai",
					"operatorName": "中国移动", "mcc": "460", "mnc": "00",
					"operator": "00", "msisdn": "", "smsc": "",
				},
				"us.json": map[string]string{
					"bs": "wifi", "category": "191", "version": "191",
					"country": "us", "language": "en-US", "timezone": "America/New_York",
					"operatorName": "T-Mobile", "mcc": "310", "mnc": "260",
					"operator": "260", "msisdn": "", "smsc": "",
				},
			}

			for name, content := range configs {
				path := fmt.Sprintf("%s/%s", dir, name)
				data, _ := json.MarshalIndent(content, "", "    ")
				if err := os.WriteFile(path, data, 0644); err != nil {
					return fmt.Errorf("写入 %s 失败: %w", name, err)
				}
				fmt.Printf("已创建: %s\n", path)
			}

			return nil
		},
	}
}

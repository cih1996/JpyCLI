package cloud

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// CloudConfig 集控平台配置
type CloudConfig struct {
	ServerURL      string             `yaml:"server_url"`
	Auth           CloudAuthConfig    `yaml:"auth"`
	LastUsedConfig string             `yaml:"last_used_config"`
	Notification   NotificationConfig `yaml:"notification"`
}

// CloudAuthConfig 集控平台认证配置
type CloudAuthConfig struct {
	SecretKey string `yaml:"secret_key"`
}

// NotificationConfig 通知配置
type NotificationConfig struct {
	LarkWebhook             string `yaml:"lark_webhook"`
	Enabled                 bool   `yaml:"enabled"`
	NotifyAlways            bool   `yaml:"notify_always"`
	OfflineThresholdPercent int    `yaml:"offline_threshold_percent"`
}

// GetCloudConfigDir 获取 cloud 配置目录
func GetCloudConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic("无法获取用户主目录: " + err.Error())
	}
	return filepath.Join(home, ".jpy", "cloud")
}

// GetCloudConfigPath 获取 cloud 配置文件路径
func GetCloudConfigPath() string {
	return filepath.Join(GetCloudConfigDir(), "config.yaml")
}

// LoadCloudConfig 加载 cloud 配置
func LoadCloudConfig() (*CloudConfig, error) {
	path := GetCloudConfigPath()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &CloudConfig{
			ServerURL: "wss://home.accjs.cn/ws",
		}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取 cloud 配置失败: %w", err)
	}

	var cfg CloudConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析 cloud 配置失败: %w", err)
	}

	if cfg.ServerURL == "" {
		cfg.ServerURL = "wss://home.accjs.cn/ws"
	}

	return &cfg, nil
}

// SaveCloudConfig 保存 cloud 配置
func SaveCloudConfig(cfg *CloudConfig) error {
	dir := GetCloudConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	return os.WriteFile(GetCloudConfigPath(), data, 0644)
}

// GetChangeOsConfigsDir 获取改机配置目录
func GetChangeOsConfigsDir() string {
	return filepath.Join(GetCloudConfigDir(), "configs")
}

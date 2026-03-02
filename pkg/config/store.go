package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// GlobalSettings holds the runtime configuration from jpy-config.yaml
var GlobalSettings Settings
var mu sync.Mutex

type Settings struct {
	LogLevel       string `yaml:"log_level"`
	LogOutput      string `yaml:"log_output"`
	MaxConcurrency int    `yaml:"max_concurrency"`
	ConnectTimeout int    `yaml:"connect_timeout"`
}

func GetConfigDir() string {
	if dir := os.Getenv("JPY_DATA_DIR"); dir != "" {
		return dir
	}

	home, err := os.UserHomeDir()
	if err != nil {
		panic("Failed to get user home directory: " + err.Error())
	}

	path := filepath.Join(home, ".jpy", "data")
	if err := os.MkdirAll(path, 0755); err != nil {
		// Just print warning, return path anyway hoping it might work or fail later
		// But in CLI context fmt.Println is acceptable for critical setup errors
	}
	return path
}

func GetConfigPath() string {
	return filepath.Join(GetConfigDir(), "config.json")
}

func GetSettingsPath() string {
	return filepath.Join(GetConfigDir(), "jpy-config.yaml")
}

func Load() (*Config, error) {
	path := GetConfigPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &Config{
			Groups: make(map[string][]LocalServerConfig),
		}, nil
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// Initialize map if nil
	if cfg.Groups == nil {
		cfg.Groups = make(map[string][]LocalServerConfig)
	}

	// Migration: Move Servers to Groups
	if len(cfg.Servers) > 0 {
		for _, s := range cfg.Servers {
			group := s.Group
			if group == "" {
				group = "default"
				s.Group = "default"
				s.Group = "default"
			}
			// Check for duplicates in the target group during migration
			exists := false
			for _, existing := range cfg.Groups[group] {
				if existing.URL == s.URL {
					exists = true
					break
				}
			}
			if !exists {
				cfg.Groups[group] = append(cfg.Groups[group], s)
			}
		}
		// Clear old list after migration
		cfg.Servers = nil
		// We could Save here, but side-effects in Load are sometimes risky.
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	mu.Lock()
	defer mu.Unlock()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(GetConfigPath(), data, 0644)
}

func UpdateServer(cfg *Config, server LocalServerConfig) error {
	group := server.Group
	if group == "" {
		group = "default"
		server.Group = "default"
	}

	servers := cfg.Groups[group]
	found := false
	for i, s := range servers {
		if s.URL == server.URL {
			servers[i] = server
			found = true
			break
		}
	}

	if !found {
		servers = append(servers, server)
	}

	cfg.Groups[group] = servers
	return Save(cfg)
}

func SetServerDisabled(cfg *Config, url string, group string, disabled bool) error {
	if group == "" {
		group = "default"
	}

	servers := cfg.Groups[group]
	for i, s := range servers {
		if s.URL == url {
			servers[i].Disabled = disabled
			cfg.Groups[group] = servers
			return Save(cfg)
		}
	}

	return fmt.Errorf("server not found: %s in group %s", url, group)
}

func RemoveServer(cfg *Config, url string, group string) error {
	if group == "" {
		group = "default"
	}

	servers := cfg.Groups[group]
	newServers := []LocalServerConfig{}
	found := false

	for _, s := range servers {
		if s.URL == url {
			found = true
			continue
		}
		newServers = append(newServers, s)
	}

	if !found {
		return fmt.Errorf("server not found: %s in group %s", url, group)
	}

	cfg.Groups[group] = newServers
	return Save(cfg)
}

func LoadSettings() *Settings {
	path := GetSettingsPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &Settings{
			LogLevel:       "info",
			LogOutput:      "stdout",
			MaxConcurrency: 50,
			ConnectTimeout: 5,
		}
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		// Return defaults on error
		return &Settings{
			LogLevel:       "info",
			LogOutput:      "stdout",
			MaxConcurrency: 50,
			ConnectTimeout: 5,
		}
	}

	var s Settings
	if err := yaml.Unmarshal(data, &s); err != nil {
		return &Settings{
			LogLevel:       "info",
			LogOutput:      "stdout",
			MaxConcurrency: 50,
			ConnectTimeout: 5,
		}
	}
	return &s
}

func SaveSettings(s *Settings) error {
	dir := GetConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(s)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(GetSettingsPath(), data, 0644)
}

func GetGroupServers(cfg *Config, group string) []LocalServerConfig {
	if group == "" || group == "all" {
		var all []LocalServerConfig
		for _, servers := range cfg.Groups {
			all = append(all, servers...)
		}
		return all
	}
	return cfg.Groups[group]
}

// AddServer adds a new server or updates an existing one
func AddServer(cfg *Config, server LocalServerConfig) error {
	return UpdateServer(cfg, server)
}

// GetAllServers returns all servers from all groups
func GetAllServers(cfg *Config) []LocalServerConfig {
	return GetGroupServers(cfg, "all")
}

// SetServerDisabledBatch sets the disabled status for multiple servers
func SetServerDisabledBatch(cfg *Config, items []BatchItem, disabled bool) error {
	mu.Lock()
	defer mu.Unlock()

	for _, item := range items {
		group := item.Group
		if group == "" {
			group = "default"
		}

		servers := cfg.Groups[group]
		for i, s := range servers {
			if s.URL == item.URL {
				servers[i].Disabled = disabled
				// Update the slice in the map
				cfg.Groups[group] = servers
				break
			}
		}
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(GetConfigPath(), data, 0644)
}

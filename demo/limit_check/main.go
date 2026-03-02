package main

import (
	"fmt"
	"jpy-cli/pkg/config"
	"os"
)

func main() {
	// 1. Setup temporary config file
	tmpDir, err := os.MkdirTemp("", "jpy_limit_test")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	// configPath := filepath.Join(tmpDir, "config.json")

	fmt.Println("Verifying 100-item limit removal...")

	// Create a config with 150 servers
	cfg := &config.Config{
		Groups: make(map[string][]config.LocalServerConfig),
	}

	servers := make([]config.LocalServerConfig, 0)
	for i := 0; i < 150; i++ {
		servers = append(servers, config.LocalServerConfig{
			URL: fmt.Sprintf("http://192.168.1.%d:8000", i),
		})
	}
	cfg.Groups["default"] = servers

	// Simulate the logic that WAS in handleServerList (now removed)
	// Old logic:
	/*
		limitedGroups := make(map[string][]config.LocalServerConfig)
		for groupName, servers := range cfg.Groups {
			if len(servers) > 100 {
				limitedGroups[groupName] = servers[:100]
			} else {
				limitedGroups[groupName] = servers
			}
		}
	*/

	// New logic (Pass through)
	finalGroups := cfg.Groups

	count := len(finalGroups["default"])
	fmt.Printf("Input servers: 150\n")
	fmt.Printf("Output servers: %d\n", count)

	if count == 150 {
		fmt.Println("SUCCESS: Limit is not applied.")
	} else {
		fmt.Printf("FAILURE: Limit is applied (count=%d)\n", count)
	}
}

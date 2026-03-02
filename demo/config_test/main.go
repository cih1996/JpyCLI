package main

import (
	"fmt"
	"jpy-cli/pkg/config"
	"log"
	"os"
	"path/filepath"
)

func main() {
	// Setup test environment
	tmpDir := filepath.Join(os.TempDir(), "jpy-test-config")
	os.Setenv("JPY_DATA_DIR", tmpDir)
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	fmt.Println("Testing Config Package in", tmpDir)

	// 1. Load Settings
	settings := config.LoadSettings()
	fmt.Printf("Loaded Settings: %+v\n", settings)

	// 2. Modify Settings via Reflection
	err := config.SetField(settings, "log_level", "debug")
	if err != nil {
		log.Fatalf("Failed to set log_level: %v", err)
	}
	fmt.Println("Set log_level to debug")

	// 3. Save Settings
	err = config.SaveSettings(settings)
	if err != nil {
		log.Fatalf("Failed to save settings: %v", err)
	}
	fmt.Println("Saved settings")

	// 4. Reload to verify
	settingsReloaded := config.LoadSettings()
	if settingsReloaded.LogLevel != "debug" {
		log.Fatalf("Verification failed: expected debug, got %s", settingsReloaded.LogLevel)
	}
	fmt.Println("Verification successful: LogLevel is debug")

	// 5. Test BatchItem and SetServerDisabledBatch
	cfg := &config.Config{
		Groups: map[string][]config.LocalServerConfig{
			"default": {
				{URL: "http://test-server:8080", Disabled: false},
			},
		},
	}

	// Save initial config so SetServerDisabledBatch has something to write
	config.Save(cfg)

	items := []config.BatchItem{
		{URL: "http://test-server:8080", Group: "default"},
	}

	err = config.SetServerDisabledBatch(cfg, items, true)
	if err != nil {
		log.Fatalf("Failed to batch disable: %v", err)
	}

	if !cfg.Groups["default"][0].Disabled {
		log.Fatal("Batch disable failed: Server is not disabled")
	}
	fmt.Println("Batch disable successful")
}

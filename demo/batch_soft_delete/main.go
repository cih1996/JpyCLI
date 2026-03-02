package main

import (
	"encoding/json"
	"fmt"
	"jpy-cli/pkg/config"
	"os"
	"path/filepath"
)

func main() {
	// 1. Setup temporary config file
	tmpDir, err := os.MkdirTemp("", "jpy-demo")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	// Set the config path for the package (we need to override the default path logic if possible,
	// but pkg/config uses a hardcoded path or relative path.
	// Let's check pkg/config/store.go to see how to override path.)
	// Looking at store.go (I recall seeing it), it might use "data/config.json".
	// Let's try to mock the environment or modify the code to accept a path?
	// Or we can just create "data/config.json" in the current directory if we run from there.

	// For this demo, let's assume we can control the config loading.
	// If pkg/config/store.go hardcodes "data/config.json", we will create that structure locally.

	cwd, _ := os.Getwd()
	fmt.Printf("Running in: %s\n", cwd)

	dataPath := filepath.Join(cwd, "data")
	os.MkdirAll(dataPath, 0755)
	defer os.RemoveAll(dataPath) // Cleanup

	// Set JPY_DATA_DIR for test
	os.Setenv("JPY_DATA_DIR", dataPath)

	initialConfig := config.Config{
		Groups: map[string][]config.LocalServerConfig{
			"default": {
				{URL: "http://192.168.1.1:8080", Disabled: false},
				{URL: "http://192.168.1.2:8080", Disabled: false},
				{URL: "http://192.168.1.3:8080", Disabled: false},
			},
		},
	}

	data, _ := json.MarshalIndent(initialConfig, "", "  ")
	os.WriteFile(filepath.Join(dataPath, "config.json"), data, 0644)

	// 2. Load Config
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}
	fmt.Println("Initial Config Loaded.")

	// 3. Batch Soft Delete
	items := []config.BatchItem{
		{URL: "http://192.168.1.1:8080", Group: "default"},
		{URL: "http://192.168.1.3:8080", Group: "default"},
	}

	fmt.Println("Disabling server 1 and 3...")
	err = config.SetServerDisabledBatch(cfg, items, true)
	if err != nil {
		fmt.Printf("Error in batch disable: %v\n", err)
		return
	}

	// 4. Verify
	// Reload config from disk to be sure
	cfg, err = config.Load()
	if err != nil {
		fmt.Printf("Error reloading config: %v\n", err)
		return
	}

	servers := cfg.Groups["default"]
	success := true
	for _, s := range servers {
		if s.URL == "http://192.168.1.1:8080" && !s.Disabled {
			fmt.Printf("FAIL: Server 1 should be disabled\n")
			success = false
		}
		if s.URL == "http://192.168.1.3:8080" && !s.Disabled {
			fmt.Printf("FAIL: Server 3 should be disabled\n")
			success = false
		}
		if s.URL == "http://192.168.1.2:8080" && s.Disabled {
			fmt.Printf("FAIL: Server 2 should NOT be disabled\n")
			success = false
		}
	}

	if success {
		fmt.Println("SUCCESS: Batch soft delete verified.")
	} else {
		fmt.Println("FAILED: Verification failed.")
	}
}

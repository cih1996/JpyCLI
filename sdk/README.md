# JPY Go SDK

This SDK provides a Go interface for interacting with JPY middleware and services.
It wraps the core functionality of the JPY CLI into an easy-to-use Go package.

## Installation

Assuming you have hosted this repository on GitHub:

```bash
go get github.com/your-username/jpy-cli-go
```

*Note: You may need to update the `go.mod` file in the root of this project to match your GitHub repository URL.*

## Usage

### Basic Example

```go
package main

import (
	"fmt"
	"log"
	
	// Import the SDK
	"jpy-cli/sdk" // Replace with your actual module path (e.g. github.com/user/repo/sdk)
)

func main() {
	// 1. Initialize the client
	// baseURL: The address of your middleware (e.g., http://127.0.0.1:8080)
	// token: Your authentication token
	client := sdk.NewClient("http://127.0.0.1:8080", "your-access-token")

	// 2. Connect to the server
	// This establishes the WebSocket connection required for real-time operations.
	if err := client.Connect(); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// 3. Use the Device API
	devices, err := client.Device.FetchDeviceList()
	if err != nil {
		log.Fatalf("Failed to list devices: %v", err)
	}

	fmt.Printf("Found %d devices:\n", len(devices))
	for _, dev := range devices {
		fmt.Printf("- %s (Status: %d)\n", dev.Serial, dev.Status)
	}

	// 4. Use Admin API (if needed)
	// captcha, err := client.Admin.GetCaptcha()
	// ...
}
```

## Features

- **Unified Client**: Single entry point for Device and Admin APIs.
- **Auto-connection**: Handles WebSocket connection setup.
- **Type-safe**: Uses strong typing for all requests and responses.

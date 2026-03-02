package sdk_test

import (
	"testing"
	"jpy-cli/sdk"
)

func TestNewClient(t *testing.T) {
	client := sdk.NewClient("http://localhost:8080", "test-token")
	if client == nil {
		t.Fatal("Expected client to be non-nil")
	}
	
	if client.Admin == nil {
		t.Error("Expected Admin client to be initialized")
	}

	// We cannot test Connect() without a real server, but we can check if the method exists
	// and handles invalid URL gracefully (mocking might be too complex for now given we just want to verify structure)
}

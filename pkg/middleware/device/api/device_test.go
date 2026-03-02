package api

import (
	"errors"
	"jpy-cli/pkg/middleware/model"
	"jpy-cli/pkg/middleware/protocol"
	"testing"
)

// MockTransport mocks the Transport interface for testing.
type MockTransport struct {
	Response *model.WSResponse
	Error    error
}

func (m *MockTransport) SendRequest(f int, data interface{}) (*model.WSResponse, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.Response, nil
}

// Ensure MockTransport implements protocol.Transport
var _ protocol.Transport = &MockTransport{}

func TestFetchDeviceList_Success(t *testing.T) {
	// Prepare mock data using map to ensure field names match expected JSON tags
	// (msgpack.Marshal of struct uses field names by default, but we expect lower-case keys matching json tags)
	mockData := []map[string]interface{}{
		{"seat": 1, "uuid": "uuid-1", "model": "Pixel 4"},
		{"seat": 2, "uuid": "uuid-2", "model": "Pixel 5"},
	}

	f := model.FuncDeviceList
	req := false
	resp := &model.WSResponse{
		F:    &f,
		Req:  &req,
		Data: mockData,
	}

	transport := &MockTransport{Response: resp}
	api := NewDeviceAPI(transport, "http://mock", "token")

	devices, err := api.FetchDeviceList()
	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	if len(devices) != 2 {
		t.Errorf("Expected 2 devices, got %d", len(devices))
	}
	if devices[0].Model != "Pixel 4" {
		t.Errorf("Expected Pixel 4, got %s", devices[0].Model)
	}
}

func TestFetchDeviceList_Error(t *testing.T) {
	transport := &MockTransport{Error: errors.New("connection failed")}
	api := NewDeviceAPI(transport, "http://mock", "token")

	_, err := api.FetchDeviceList()
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if err.Error() != "connection failed" {
		t.Errorf("Expected 'connection failed', got '%v'", err)
	}
}

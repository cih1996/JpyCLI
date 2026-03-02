package protocol

import (
	"jpy-cli/pkg/middleware/model"
)

// Transport defines the interface for sending requests to the server.
// This decouples the API layer from the specific transport implementation (WebSocket, HTTP, etc.).
type Transport interface {
	SendRequest(f int, data interface{}) (*model.WSResponse, error)
}

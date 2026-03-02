package ws

import (
	"encoding/json"
	"fmt"
	"jpy-cli/pkg/logger"
	"net/http"

	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local CLI tool
	},
}

type SafeConn struct {
	*websocket.Conn
	mu sync.Mutex
}

func (c *SafeConn) WriteJSON(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Conn.WriteJSON(v)
}

func (c *SafeConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Conn.Close()
}

type Server struct {
	addr string
}

func NewServer(addr string) *Server {
	return &Server{addr: addr}
}

func (s *Server) Start() error {
	http.HandleFunc("/ws", s.HandleWebSocket)

	// Serve static files will be handled by a separate handler or added here later

	logger.Infof("Starting WebSocket server on %s", s.addr)
	return http.ListenAndServe(s.addr, nil)
}

type Request struct {
	ID      string          `json:"id"`
	Command string          `json:"command"`
	Params  json.RawMessage `json:"params"`
}

type Response struct {
	ID     string `json:"id"`
	Status string `json:"status"` // "success" or "error"
	Data   any    `json:"data,omitempty"`
	Error  string `json:"error,omitempty"`
}

// HandleWebSocket exports the websocket handler
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Errorf("Failed to upgrade to WebSocket: %v", err)
		return
	}

	safeConn := &SafeConn{Conn: conn}

	defer func() {
		safeConn.Close()
		fmt.Printf("[WS] Connection closed from %s\n", r.RemoteAddr)
	}()

	logger.Infof("New WebSocket connection from %s", r.RemoteAddr)
	fmt.Printf("[WS] Connection from %s\n", r.RemoteAddr)

	for {
		_, message, err := safeConn.ReadMessage()
		if err != nil {
			logger.Errorf("WebSocket read error: %v", err)
			fmt.Printf("[WS] Read error: %v\n", err)
			break
		}

		// Log received message (limited length to avoid spamming terminal too much if huge)
		msgStr := string(message)
		if len(msgStr) > 500 {
			fmt.Printf("[WS] Received: %s... (truncated)\n", msgStr[:500])
		} else {
			fmt.Printf("[WS] Received: %s\n", msgStr)
		}

		var req Request
		if err := json.Unmarshal(message, &req); err != nil {
			sendError(safeConn, "", fmt.Sprintf("Invalid JSON format: %v", err))
			continue
		}

		// Handle request concurrently
		go s.handleRequest(safeConn, req)
	}
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	s.HandleWebSocket(w, r)
}

func sendError(conn *SafeConn, id, msg string) {
	resp := Response{
		ID:     id,
		Status: "error",
		Error:  msg,
	}
	conn.WriteJSON(resp)
}

func sendSuccess(conn *SafeConn, id string, data interface{}) {
	resp := Response{
		ID:     id,
		Status: "success",
		Data:   data,
	}
	conn.WriteJSON(resp)
}

// Handler registry
type HandlerFunc func(*SafeConn, Request)

var handlers = map[string]HandlerFunc{}

func RegisterHandler(command string, handler HandlerFunc) {
	handlers[command] = handler
}

func (s *Server) handleRequest(conn *SafeConn, req Request) {
	fmt.Printf("[WS] Handling command: %s (ID: %s)\n", req.Command, req.ID)
	handler, ok := handlers[req.Command]
	if !ok {
		sendError(conn, req.ID, fmt.Sprintf("Unknown command: %s", req.Command))
		return
	}

	// Recover from panics in handlers
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("Panic in handler %s: %v", req.Command, r)
			sendError(conn, req.ID, fmt.Sprintf("Internal server error: %v", r))
		}
	}()

	handler(conn, req)
}

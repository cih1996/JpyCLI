package server

import (
	"crypto/tls"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"jpy-cli/internal/server/ws"

	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

//go:embed static/*
var content embed.FS

var (
	proxyStore sync.Map // map[token]targetURL
)

const (
	cookieName = "jpy-proxy-target"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity in this context
	},
}

func NewWebServerCmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "web",
		Short: "启动 Web 终端服务 (推荐)",
		Long: `启动一个基于 Web 的终端服务，提供比 SSH 更优秀的 TUI 兼容性和跨平台体验。
无需安装任何客户端，直接通过浏览器访问即可。

特性:
- 完美支持 Windows/macOS/Linux
- 自动适应窗口大小
- 支持鼠标操作和所有快捷键
- 解决 Windows 下 SSH 终端渲染错位问题
- 支持自定义快捷命令侧边栏`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initialize WebSocket Server Handler
			wsServer := ws.NewServer("")
			http.HandleFunc("/ws/api", wsServer.HandleWebSocket)

			// Serve static files
			// Embed path is "static", so we need to strip prefix if we want /static/x.js
			// But for root index.html:

			// Main Handler
			http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				// 1. Check for Proxy Exit
				if r.URL.Path == "/jpy-exit" {
					// Clear global proxy target
					setGlobalProxyTarget("")

					http.SetCookie(w, &http.Cookie{
						Name:   cookieName,
						Value:  "",
						Path:   "/",
						MaxAge: -1,
					})
					// Clear HSTS
					w.Header().Set("Strict-Transport-Security", "max-age=0")
					http.Redirect(w, r, "/", http.StatusFound)
					return
				}

				// 2. Check for Proxy Activation /go/:token
				if len(r.URL.Path) > 4 && r.URL.Path[:4] == "/go/" {
					token := r.URL.Path[4:]
					if target, ok := proxyStore.Load(token); ok {
						targetStr := target.(string)

						// Set Global Proxy Target if it's HTTPS (or we want to force TCP mode)
						targetURL, _ := url.Parse(targetStr)
						if targetURL.Scheme == "https" {
							// For HTTPS, we use the TCP Forwarding mode (Protocol Sniffing)
							// We need host:port
							host := targetURL.Host
							if targetURL.Port() == "" {
								host += ":443"
							}
							setGlobalProxyTarget(host)
						} else {
							// For HTTP, we can stick to ReverseProxy (better control)
							// Or we can also use TCP forwarding if we want?
							// Let's stick to ReverseProxy for HTTP to allow easier exit?
							// User said "regardless of data, forward to target"
							// But for HTTP, ReverseProxy is fine.
							// However, to be consistent with user request "turn 8080 into another mode",
							// maybe we should just set it for everything?
							// But if we set it for HTTP, we lose /jpy-exit unless we sniff HTTP too.
							// Current sniffer only checks 0x16 (HTTPS).
							// So HTTP traffic still comes here.
							// So for HTTP targets, we don't set global TCP target, we use Cookie.
							setGlobalProxyTarget("") // Clear it just in case
						}

						// Set Cookie (still needed for HTTP mode)
						http.SetCookie(w, &http.Cookie{
							Name:  cookieName,
							Value: targetStr,
							Path:  "/",
						})
						// Remove token to make it one-time use (optional, keeping it for now allows refresh)
						// proxyStore.Delete(token)

						// Check if target is HTTPS and we set the global proxy target
						if targetURL.Scheme == "https" {
							// If we are currently on HTTP (which is likely if accessing via IP:Port directly)
							// We should try to redirect to HTTPS to trigger the TCP forwarding
							// But we don't know if the user can access HTTPS on this port (e.g. if frpc supports it)
							// Assuming user setup allows it (as per requirement "reusing port")

							// Check if current request is already HTTPS (unlikely here as we are in HTTP handler)
							if r.TLS == nil {
								// Construct HTTPS URL
								httpsURL := "https://" + r.Host + "/"

								// Clear HSTS to avoid browser caching HTTPS for this domain
								w.Header().Set("Strict-Transport-Security", "max-age=0")
								http.Redirect(w, r, httpsURL, http.StatusFound)
								return
							}
						}

						// Clear HSTS here too
						w.Header().Set("Strict-Transport-Security", "max-age=0")
						http.Redirect(w, r, "/", http.StatusFound)
						return
					}
					http.Error(w, "Invalid or expired proxy token", http.StatusNotFound)
					return
				}

				// 3. Check for Active Proxy Cookie
				cookie, err := r.Cookie(cookieName)
				if err == nil && cookie.Value != "" {
					targetURL, err := url.Parse(cookie.Value)
					if err == nil {
						proxy := httputil.NewSingleHostReverseProxy(targetURL)

						// 1. Configure Transport to support self-signed certs
						proxy.Transport = &http.Transport{
							TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
						}

						// 2. Customize Director to handle WebSocket upgrades
						originalDirector := proxy.Director
						proxy.Director = func(req *http.Request) {
							originalDirector(req)
							req.Host = targetURL.Host

							// Check for WebSocket Upgrade
							if req.Header.Get("Upgrade") != "" && req.Header.Get("Connection") != "" {
								// Fix Scheme for WebSocket
								if targetURL.Scheme == "https" {
									req.URL.Scheme = "wss"
								} else {
									req.URL.Scheme = "ws"
								}
							}
						}

						// 3. Error Handler
						proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
							fmt.Printf("[Proxy Error] %v\n", err)
							http.Error(w, fmt.Sprintf("Proxy Error: %v", err), http.StatusBadGateway)
						}

						proxy.ServeHTTP(w, r)
						return
					}
				}

				// 4. Default: Web Terminal
				if r.URL.Path == "/dashboard" {
					data, _ := content.ReadFile("static/dashboard.html")
					w.Header().Set("Content-Type", "text/html")
					w.Write(data)
					return
				}

				if r.URL.Path == "/" || r.URL.Path == "/index.html" {
					data, _ := content.ReadFile("static/index.html")
					w.Write(data)
					return
				}
				// Serve other static assets if any (not needed for now as we use CDN for xterm)
			})

			http.HandleFunc("/ws", handleWebsocket)

			// Create Proxy Token API
			http.HandleFunc("/sys/proxy/create", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}

				var req struct {
					Target string `json:"target"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}

				if req.Target == "" {
					http.Error(w, "Target URL is required", http.StatusBadRequest)
					return
				}

				// Generate Token
				token := generateToken(8)
				proxyStore.Store(token, req.Target)

				// Cleanup token after 1 hour (simple cleanup)
				go func() {
					time.Sleep(1 * time.Hour)
					proxyStore.Delete(token)
				}()

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{
					"token": token,
				})
			})

			http.HandleFunc("/config/commands", func(w http.ResponseWriter, r *http.Request) {
				data, err := content.ReadFile("static/quick_commands.json")
				if err != nil {
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte("[]"))
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.Write(data)
			})

			addr := fmt.Sprintf(":%d", port)

			// Create Listener
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				return err
			}

			fmt.Printf("Web Terminal started at http://localhost%s\n", addr)
			fmt.Printf("Press Ctrl+C to stop.\n")

			// Custom Handler that handles both HTTP and TCP sniffing
			// Use a custom loop to avoid logging errors for hijacked connections
			server := &http.Server{Handler: nil} // nil handler uses DefaultServeMux
			return server.Serve(&sniffListener{Listener: ln})
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 8080, "Web 服务监听端口")
	return cmd
}

type wsMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

// sniffListener wraps a net.Listener to peek at the first byte of connections
type sniffListener struct {
	net.Listener
}

func (l *sniffListener) Accept() (net.Conn, error) {
	for {
		c, err := l.Listener.Accept()
		if err != nil {
			return nil, err
		}

		// Read first byte to sniff protocol
		// We need a buffer that allows peeking
		peekConn := &peekedConn{Conn: c}
		// Peek 1 byte
		peekBuf := make([]byte, 1)
		n, err := c.Read(peekBuf)
		if err != nil {
			c.Close()
			// If read error, just continue to next connection
			continue
		}
		peekConn.peeked = peekBuf[:n]

		// Check if it's a TLS Client Hello (First byte 0x16)
		// AND we have an active proxy target
		if n > 0 && peekBuf[0] == 0x16 {
			if target, ok := getGlobalProxyTarget(); ok {
				// Transparent TCP Forwarding
				go handleTCPForward(peekConn, target)

				// CRITICAL FIX: Do NOT return error here.
				// Returning error causes http.Server to stop.
				// Instead, we just loop back and accept the next connection.
				// This connection is now handled by handleTCPForward in a goroutine.
				continue
			}
		}

		return peekConn, nil
	}
}

// Global Proxy State
var (
	globalProxyTarget string
	globalProxyMutex  sync.RWMutex
)

func setGlobalProxyTarget(target string) {
	globalProxyMutex.Lock()
	defer globalProxyMutex.Unlock()
	globalProxyTarget = target
}

func getGlobalProxyTarget() (string, bool) {
	globalProxyMutex.RLock()
	defer globalProxyMutex.RUnlock()
	return globalProxyTarget, globalProxyTarget != ""
}

func handleTCPForward(src net.Conn, targetAddr string) {
	defer src.Close()

	// Connect to target
	dst, err := net.Dial("tcp", targetAddr)
	if err != nil {
		fmt.Printf("Failed to dial target %s: %v\n", targetAddr, err)
		return
	}
	defer dst.Close()

	// Bidirectional copy
	go io.Copy(dst, src)
	io.Copy(src, dst)
}

type peekedConn struct {
	net.Conn
	peeked []byte
}

func (c *peekedConn) Read(p []byte) (n int, err error) {
	if len(c.peeked) > 0 {
		n = copy(p, c.peeked)
		c.peeked = c.peeked[n:]
		if n == len(p) {
			return n, nil
		}
		// Continue reading from underlying conn
		m, err := c.Conn.Read(p[n:])
		return n + m, err
	}
	return c.Conn.Read(p)
}

func handleWebsocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// 1. Prepare Command
	cmd := exec.Command(getUserShell())
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	// Ensure jpy executable is in PATH (same logic as SSH)
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		currentPath := os.Getenv("PATH")
		newPath := fmt.Sprintf("PATH=%s%c%s", exeDir, os.PathListSeparator, currentPath)
		cmd.Env = append(cmd.Env, newPath)
	}

	// 2. Start PTY
	ptyIO, err := StartPTY(cmd)
	if err != nil {
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Failed to start PTY: %v", err)))
		return
	}
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		ptyIO.Close()
	}()

	// 3. Output Loop (PTY -> WebSocket)
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := ptyIO.Read(buf)
			if err != nil {
				break
			}
			// Send as Binary to avoid UTF-8 validation issues in some browsers/libs
			// although TextMessage usually works fine for VT100
			err = conn.WriteMessage(websocket.BinaryMessage, buf[:n])
			if err != nil {
				break
			}
		}
		conn.Close()
	}()

	// 4. Input Loop (WebSocket -> PTY)
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg wsMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "input":
			ptyIO.Write([]byte(msg.Data))
		case "resize":
			ResizePTY(ptyIO, msg.Cols, msg.Rows)
		}
	}
}

func generateToken(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

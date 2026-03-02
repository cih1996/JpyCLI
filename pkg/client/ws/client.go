package wsclient

import (
	"bytes"
	"crypto/tls"
	"errors"
	"jpy-cli/pkg/logger"
	"jpy-cli/pkg/middleware/model"
	"jpy-cli/pkg/middleware/protocol"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vmihailenco/msgpack/v5"
)

type Client struct {
	URL      string
	Endpoint string // e.g. "/box/subscribe" or "/box/mirror"
	Params   map[string]string
	Token    string
	Conn     *websocket.Conn
	Timeout  time.Duration

	// Concurrency control
	sendMu sync.Mutex
	done   chan struct{}

	// Request correlation
	seq     uint32
	pending sync.Map // map[uint32]chan *model.WSResponse

	// Event handlers
	OnMessage func(msgType int, data []byte)
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		URL:   baseURL,
		Token: token,
		done:  make(chan struct{}),
	}
}

// SendRaw sends a raw message without encoding
func (c *Client) SendRaw(data []byte) error {
	if c.Conn == nil {
		return errors.New("not connected")
	}
	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	return c.Conn.WriteMessage(websocket.BinaryMessage, data)
}

func (c *Client) Connect() error {
	u, err := url.Parse(c.URL)
	if err != nil {
		return err
	}

	scheme := "ws"
	if u.Scheme == "https" {
		scheme = "wss"
	}
	u.Scheme = scheme

	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = "/box/subscribe"
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + endpoint

	q := u.Query()
	q.Set("Authorization", c.Token)
	for k, v := range c.Params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	logger.Log.Debug("正在连接 WebSocket", "url", u.String())

	dialer := websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	if c.Timeout > 0 {
		dialer.HandshakeTimeout = c.Timeout
	} else {
		dialer.HandshakeTimeout = 10 * time.Second
	}

	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		logger.Log.Error("WebSocket 连接失败", "error", err)
		return err
	}
	c.Conn = conn

	// Start read loop
	go c.readLoop()

	// Send Init (System Sync) only for Subscribe channel
	if endpoint == "/box/subscribe" {
		if err := c.sendInit(); err != nil {
			c.Close()
			return err
		}
	}

	return nil
}

func (c *Client) Close() {
	select {
	case <-c.done:
		return // 已关闭
	default:
		close(c.done)
	}
	if c.Conn != nil {
		c.Conn.Close()
	}
}

func (c *Client) readLoop() {
	defer c.Close()
	for {
		select {
		case <-c.done:
			return
		default:
			// Set read deadline to detect connection loss
			c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			_, message, err := c.Conn.ReadMessage()
			if err != nil {
				if !strings.Contains(err.Error(), "use of closed network connection") {
					logger.Log.Debug("读取消息失败", "error", err)
				}
				return
			}

			// Use Unpack to get raw body
			msgType, _, body, err := protocol.Unpack(message)
			if err != nil {
				logger.Log.Debug("解包消息失败", "error", err)
				continue
			}

			// Notify handler if set
			if c.OnMessage != nil {
				c.OnMessage(msgType, body)
			}

			if msgType == protocol.TypeMsgpack {
				var resp model.WSResponse
				dec := msgpack.NewDecoder(bytes.NewReader(body))
				dec.SetCustomStructTag("json")
				if err := dec.Decode(&resp); err == nil {
					// Check for Seq match
					if resp.Seq != nil {
						seq := uint32(*resp.Seq)
						if ch, ok := c.pending.Load(seq); ok {
							select {
							case ch.(chan *model.WSResponse) <- &resp:
							default:
								logger.Log.Warn("响应通道已满，丢弃消息", "seq", seq)
							}
						}
					}
					// TODO: Handle push messages (Seq=0 or nil) via an event handler
				}
			}
		}
	}
}

func (c *Client) SendRequest(f int, data interface{}) (*model.WSResponse, error) {
	if c.Conn == nil {
		return nil, errors.New("未连接")
	}

	seq := atomic.AddUint32(&c.seq, 1)
	req := model.WSRequest{
		F:    f,
		Req:  true,
		Seq:  int(seq),
		Data: data,
	}

	encoded, err := protocol.Encode(req, protocol.TypeMsgpack, []uint64{0})
	if err != nil {
		return nil, err
	}

	// Register channel
	ch := make(chan *model.WSResponse, 1)
	c.pending.Store(seq, ch)
	defer c.pending.Delete(seq)

	c.sendMu.Lock()
	err = c.Conn.WriteMessage(websocket.BinaryMessage, encoded)
	c.sendMu.Unlock()
	if err != nil {
		return nil, err
	}

	// Wait for response
	timeout := 10 * time.Second
	if c.Timeout > 0 {
		timeout = c.Timeout
	}

	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(timeout):
		return nil, errors.New("等待响应超时")
	case <-c.done:
		return nil, errors.New("连接已关闭")
	}
}

func (c *Client) sendInit() error {
	// System Sync doesn't necessarily need a response waited on here,
	// but we can use SendRequest if we want to confirm it.
	// For now, keep it fire-and-forget or just basic write to match previous behavior logic
	// but using the new structure.
	// Actually, Init is usually a handshake. Let's just write it.

	req := model.WSRequest{
		F:   model.FuncSystemSync,
		Req: true,
		Seq: int(atomic.AddUint32(&c.seq, 1)),
	}
	data, err := protocol.Encode(req, protocol.TypeMsgpack, []uint64{0})
	if err != nil {
		return err
	}

	c.sendMu.Lock()
	defer c.sendMu.Unlock()
	return c.Conn.WriteMessage(websocket.BinaryMessage, data)
}

package terminal

import (
	"encoding/binary"
	"errors"
	"jpy-cli/pkg/client/ws"
	"jpy-cli/pkg/logger"
	"jpy-cli/pkg/middleware/model"
	"jpy-cli/pkg/middleware/protocol"
	"time"
)

type TerminalSession struct {
	Client   *wsclient.Client
	DeviceID int64
	Ready    chan struct{}
	Output   chan string
	Closed   chan struct{}
}

func NewTerminalSession(client *wsclient.Client, deviceID int64) *TerminalSession {
	t := &TerminalSession{
		Client:   client,
		DeviceID: deviceID,
		Ready:    make(chan struct{}),
		Output:   make(chan string, 100),
		Closed:   make(chan struct{}),
	}

	client.OnMessage = t.handleMessage
	return t
}

func (t *TerminalSession) Init() error {
	// Send Terminal Init Request (f=9)
	req := map[string]interface{}{
		"action": 1,
		"rows":   36,
		"cols":   120,
	}
	
	wsReq := model.WSRequest{
		F:    model.FuncTerminalInit, // 9
		Req:  true,
		Seq:  0,
		Data: req,
	}

	encoded, err := protocol.Encode(wsReq, protocol.TypeMsgpack, []uint64{uint64(t.DeviceID)})
	if err != nil {
		return err
	}

	logger.Log.Debug("Sending Terminal Init", "deviceID", t.DeviceID)
	return t.Client.SendRaw(encoded)
}

func (t *TerminalSession) WaitForReady(timeout time.Duration) error {
	select {
	case <-t.Ready:
		return nil
	case <-time.After(timeout):
		return errors.New("timeout waiting for terminal ready")
	case <-t.Closed:
		return errors.New("terminal connection closed")
	}
}

func (t *TerminalSession) Exec(cmd string) error {
	// Protocol: Type(13) + HeaderLen(8) + DeviceID(8 bytes LE) + Data
	// Data is the command string (bytes)

	cmdBytes := []byte(cmd + "\n")
	totalLen := 1 + 1 + 8 + len(cmdBytes)
	
	buf := make([]byte, totalLen)
	buf[0] = 13 // Type Terminal Data
	buf[1] = 8  // Header Len
	
	binary.LittleEndian.PutUint64(buf[2:], uint64(t.DeviceID))
	copy(buf[10:], cmdBytes)

	logger.Log.Debug("Sending Terminal Command", "cmd", cmd)
	return t.Client.SendRaw(buf)
}

func (t *TerminalSession) Close() {
	select {
	case <-t.Closed:
		return
	default:
		close(t.Closed)
		t.Client.Close()
	}
}

func (t *TerminalSession) handleMessage(msgType int, data []byte) {
	if msgType == 13 || msgType == protocol.TypeMsgpack { // Sometimes it might be wrapped? Assuming 13.
		text := string(data)
		
		// Check for Ready signal '$'
		select {
		case <-t.Ready:
			// Already ready, just stream output
			select {
			case t.Output <- text:
			default:
				// Drop if buffer full? Or maybe increase buffer.
			}
		default:
			// Not ready yet, check for '$'
			select {
			case t.Output <- text:
			default:
			}

			// Simple check for '$' or '#'
			for _, b := range data {
				if b == '$' || b == '#' { 
					close(t.Ready)
					break
				}
			}
		}
	}
}

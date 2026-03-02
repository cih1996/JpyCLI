package protocol

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"jpy-cli/pkg/middleware/model"

	"github.com/vmihailenco/msgpack/v5"
)

const (
	TypeMsgpack = 6
	TypeJSON    = 7
)

func Encode(data interface{}, msgType int, deviceIds []uint64) ([]byte, error) {
	var body []byte
	var err error

	if msgType == TypeMsgpack {
		var buf bytes.Buffer
		enc := msgpack.NewEncoder(&buf)
		enc.SetCustomStructTag("json")
		err = enc.Encode(data)
		body = buf.Bytes()
	} else if msgType == TypeJSON {
		body, err = json.Marshal(data)
	} else {
		return nil, fmt.Errorf("unsupported message type: %d", msgType)
	}

	if err != nil {
		return nil, err
	}

	headerLen := len(deviceIds) * 8
	if headerLen > 240 {
		return nil, errors.New("too many device IDs")
	}

	totalLen := 1 + 1 + headerLen + len(body)
	buf := new(bytes.Buffer)
	buf.Grow(totalLen)

	buf.WriteByte(byte(msgType))
	buf.WriteByte(byte(headerLen))

	for _, id := range deviceIds {
		binary.Write(buf, binary.LittleEndian, id)
	}

	buf.Write(body)
	return buf.Bytes(), nil
}

// Unpack extracts the parts of the message without decoding the body
func Unpack(data []byte) (msgType int, deviceIds []uint64, body []byte, err error) {
	if len(data) < 2 {
		return 0, nil, nil, errors.New("data too short")
	}

	msgType = int(data[0])
	headerLen := int(data[1])
	offset := 2 + headerLen

	if len(data) < offset {
		return 0, nil, nil, errors.New("incomplete header")
	}

	if headerLen > 0 {
		count := headerLen / 8
		deviceIds = make([]uint64, count)
		r := bytes.NewReader(data[2 : 2+headerLen])
		if err := binary.Read(r, binary.LittleEndian, &deviceIds); err != nil {
			return 0, nil, nil, err
		}
	}

	body = data[offset:]
	return msgType, deviceIds, body, nil
}

func Decode(data []byte) (*model.WSResponse, error) {
	msgType, _, body, err := Unpack(data)
	if err != nil {
		return nil, err
	}

	var resp model.WSResponse

	if msgType == TypeMsgpack {
		dec := msgpack.NewDecoder(bytes.NewReader(body))
		dec.SetCustomStructTag("json")
		err := dec.Decode(&resp)
		if err != nil {
			// Try JSON fallback
			if jsonErr := json.Unmarshal(body, &resp); jsonErr == nil {
				return &resp, nil
			}
			return nil, err
		}
	} else if msgType == TypeJSON {
		err := json.Unmarshal(body, &resp)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("unknown message type: %d", msgType)
	}

	return &resp, nil
}

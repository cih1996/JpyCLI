package test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"jpy-cli/pkg/middleware/model"
	"jpy-cli/pkg/middleware/protocol"

	"github.com/spf13/cobra"
	"github.com/vmihailenco/msgpack/v5"
)

func NewUnpackCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "unpack [hex-string]",
		Short: "Unpack and decode a hex-encoded message",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hexStr := args[0]
			data, err := hex.DecodeString(hexStr)
			if err != nil {
				return fmt.Errorf("invalid hex string: %v", err)
			}

			msgType, deviceIds, body, err := protocol.Unpack(data)
			if err != nil {
				return fmt.Errorf("unpack failed: %v", err)
			}

			fmt.Printf("Message Type: %d\n", msgType)
			if len(deviceIds) > 0 {
				fmt.Printf("Device IDs: %v\n", deviceIds)
			}
			fmt.Printf("Body Length: %d bytes\n", len(body))

			if msgType == protocol.TypeMsgpack {
				// Try WSResponse first
				var resp model.WSResponse
				dec := msgpack.NewDecoder(bytes.NewReader(body))
				dec.SetCustomStructTag("json")
				if err := dec.Decode(&resp); err == nil {
					jsonBytes, _ := json.MarshalIndent(resp, "", "  ")
					fmt.Println("Decoded Body (WSResponse):")
					fmt.Println(string(jsonBytes))
					return nil
				}

				// Fallback to generic map
				var generic interface{}
				decGeneric := msgpack.NewDecoder(bytes.NewReader(body))
				decGeneric.SetCustomStructTag("json")
				if err := decGeneric.Decode(&generic); err == nil {
					jsonBytes, _ := json.MarshalIndent(generic, "", "  ")
					fmt.Println("Decoded Body (Generic):")
					fmt.Println(string(jsonBytes))
					return nil
				}

				return fmt.Errorf("failed to decode msgpack body")
			} else if msgType == protocol.TypeJSON {
				var decoded interface{}
				if err := json.Unmarshal(body, &decoded); err != nil {
					fmt.Println("Body (Raw JSON):")
					fmt.Println(string(body))
				} else {
					jsonBytes, _ := json.MarshalIndent(decoded, "", "  ")
					fmt.Println("Decoded Body (JSON):")
					fmt.Println(string(jsonBytes))
				}
			} else {
				fmt.Printf("Unknown message type body: %x\n", body)
			}

			return nil
		},
	}
}

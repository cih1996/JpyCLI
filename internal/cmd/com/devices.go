package com

import (
	"encoding/json"
	"fmt"
	"jpy-cli/pkg/comport"

	"github.com/spf13/cobra"
)

func NewDevicesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "devices",
		Short: "获取设备通道状态",
		RunE: func(cmd *cobra.Command, args []string) error {
			port, err := comport.ResolvePort(flagPort, "com devices")
			if err != nil {
				return err
			}

			result, err := comport.RunDevices(port, false, flagSkipConnect)
			if err != nil {
				return err
			}

			switch flagOutput {
			case "json":
				b, _ := json.Marshal(result)
				fmt.Println(string(b))
			default:
				fmt.Printf("PORT\tUID\tVERSION\tMAC\tIP\n")
				fmt.Printf("%s\t%s\t%s\t%s\t%s\n", result.Port, result.UID, result.Version, result.MAC, result.IP)
				fmt.Printf("CH\tPLUG\tMODE\n")
				for _, ch := range result.Channels {
					fmt.Printf("%d\t%s\t%s\n", ch.Channel, ch.Plug, ch.Mode)
				}
			}
			return nil
		},
	}
}

package com

import (
	"encoding/json"
	"fmt"
	"jpy-cli/pkg/comport"

	"github.com/spf13/cobra"
)

func NewListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列举所有可用 COM 口",
		RunE: func(cmd *cobra.Command, args []string) error {
			ports, err := comport.ListPorts()
			if err != nil {
				return err
			}

			switch flagOutput {
			case "json":
				b, _ := json.Marshal(map[string]interface{}{"ports": ports})
				fmt.Println(string(b))
			default:
				for _, p := range ports {
					fmt.Println(p)
				}
			}
			return nil
		},
	}
}

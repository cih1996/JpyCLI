package com

import (
	"github.com/spf13/cobra"
)

var (
	flagPort        string
	flagSkipConnect bool
	flagOutput      string
)

func NewComCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "com",
		Short: "COM 串口设备管理命令",
	}

	cmd.PersistentFlags().StringVar(&flagPort, "port", "", "串口名称（如 COM3、/dev/cu.wchusbserial124230），不指定则自动选择")
	cmd.PersistentFlags().BoolVar(&flagSkipConnect, "skip-connect", false, "跳过 0x02 建连指令")
	cmd.PersistentFlags().StringVarP(&flagOutput, "output", "o", "plain", "输出模式: plain/json")

	cmd.AddCommand(NewListCmd())
	cmd.AddCommand(NewDevicesCmd())
	cmd.AddCommand(NewSetModeCmd())
	cmd.AddCommand(NewRestartCmd())

	return cmd
}

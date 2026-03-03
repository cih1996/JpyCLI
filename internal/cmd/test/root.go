package test

import "github.com/spf13/cobra"

func NewTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "测试与调试工具",
	}

	cmd.AddCommand(NewUnpackCmd())
	return cmd
}

package middleware

import (
	"fmt"
	"jpy-cli/pkg/admin-middleware/api"
	"jpy-cli/pkg/admin-middleware/service"
	"strings"

	"github.com/spf13/cobra"
)

func NewGetRootPasswordCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get-root-password [key]",
		Short: "获取 Root 密码",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// 1. Ensure Login
			adminCfg, err := service.EnsureOperationLoggedIn()
			if err != nil {
				return err
			}

			client := api.NewClient(adminCfg.Token)
			key := strings.TrimSpace(args[0])

			// 2. Call API
			password, err := client.DecryptPassword(key)
			if err != nil {
				return fmt.Errorf("获取密码失败: %v", err)
			}

			// 3. Output
			fmt.Println(password)
			return nil
		},
	}

	return cmd
}

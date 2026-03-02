package auth

import (
	"fmt"
	httpclient "jpy-cli/pkg/client/http"
	"jpy-cli/pkg/config"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func NewLoginCmd() *cobra.Command {
	var username, password, group string

	cmd := &cobra.Command{
		Use:   "login [url]",
		Short: "登录 JPY 服务器",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			url := args[0]
			if !strings.HasPrefix(url, "http") {
				url = "https://" + url
			}

			if username == "" || password == "" {
				fmt.Println("必须提供用户名和密码")
				os.Exit(1)
			}

			client := httpclient.NewClient(url, "")
			token, err := client.Login(username, password)
			if err != nil {
				fmt.Printf("登录失败: %v\n", err)
				os.Exit(1)
			}

			cfg, err := config.Load()
			if err != nil {
				fmt.Printf("无法加载配置: %v\n", err)
				os.Exit(1)
			}

			if group == "" {
				group = "default"
			}

			server := config.LocalServerConfig{
				URL:      url,
				Username: username,
				Password: password,
				Group:    group,
				Token:    token,
			}
			config.AddServer(cfg, server)

			if err := config.Save(cfg); err != nil {
				fmt.Printf("无法保存配置: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("登录成功! 已保存到分组 '%s'。\n", group)
		},
	}

	cmd.Flags().StringVarP(&username, "username", "u", "", "用户名")
	cmd.Flags().StringVarP(&password, "password", "p", "", "密码")
	cmd.Flags().StringVarP(&group, "group", "g", "default", "客户分组名称")

	return cmd
}

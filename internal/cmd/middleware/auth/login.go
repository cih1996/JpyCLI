package auth

import (
	"encoding/json"
	"fmt"
	httpclient "jpy-cli/pkg/client/http"
	"jpy-cli/pkg/config"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func NewLoginCmd() *cobra.Command {
	var username, password, group, outputMode string

	cmd := &cobra.Command{
		Use:   "login [url]",
		Short: "登录 JPY 服务器",
		Long: `登录 JPY 中间件服务器并保存凭证。

输出模式:
  --output tui     默认输出
  --output plain   纯文本，适合脚本
  --output json    JSON 格式，适合程序解析`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			if !strings.HasPrefix(url, "http") {
				url = "https://" + url
			}

			if username == "" || password == "" {
				return fmt.Errorf("必须提供用户名和密码")
			}

			client := httpclient.NewClient(url, "")
			token, loginErr := client.Login(username, password)

			// JSON 模式
			if outputMode == "json" {
				result := map[string]interface{}{
					"url":     cleanAuthURL(url),
					"group":   group,
					"success": loginErr == nil,
				}
				if loginErr != nil {
					result["error"] = loginErr.Error()
				}
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(data))
				if loginErr != nil {
					os.Exit(1)
				}
			} else if outputMode == "plain" {
				// Plain 模式
				if loginErr != nil {
					fmt.Printf("FAILED\t%s\t%s\n", cleanAuthURL(url), loginErr.Error())
					os.Exit(1)
				}
				fmt.Printf("OK\t%s\t%s\n", cleanAuthURL(url), group)
			} else {
				// TUI 模式
				if loginErr != nil {
					return fmt.Errorf("登录失败: %v", loginErr)
				}
			}

			if loginErr != nil {
				return nil
			}

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("无法加载配置: %v", err)
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
				return fmt.Errorf("无法保存配置: %v", err)
			}

			if outputMode == "tui" || outputMode == "" {
				fmt.Printf("登录成功! 已保存到分组 '%s'。\n", group)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&username, "username", "u", "", "用户名")
	cmd.Flags().StringVarP(&password, "password", "p", "", "密码")
	cmd.Flags().StringVarP(&group, "group", "g", "default", "客户分组名称")
	cmd.Flags().StringVarP(&outputMode, "output", "o", "tui", "输出模式 (tui/plain/json)")

	return cmd
}

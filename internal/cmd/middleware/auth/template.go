package auth

import (
	"encoding/json"
	"fmt"
	"jpy-cli/pkg/config"

	"github.com/spf13/cobra"
)

func NewTemplateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "template",
		Short: "生成服务器配置模板",
		Run: func(cmd *cobra.Command, args []string) {
			template := []config.LocalServerConfig{
				{
					URL:      "https://192.168.x.x",
					Username: "admin",
					Password: "password",
					Group:    "default",
				},
			}

			data, _ := json.MarshalIndent(template, "", "  ")
			fmt.Println(string(data))
			fmt.Println("\n# 保存到文件(例如 servers.json)后使用 import 命令导入")
		},
	}
}

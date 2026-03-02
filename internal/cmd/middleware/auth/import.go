package auth

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"jpy-cli/pkg/config"
	"os"

	"github.com/spf13/cobra"
)

func NewImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import [file]",
		Short: "从 JSON 文件导入服务器配置",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			filePath := args[0]
			data, err := ioutil.ReadFile(filePath)
			if err != nil {
				fmt.Printf("读取文件失败: %v\n", err)
				os.Exit(1)
			}

			var servers []config.LocalServerConfig
			if err := json.Unmarshal(data, &servers); err != nil {
				fmt.Printf("解析 JSON 失败: %v\n", err)
				os.Exit(1)
			}

			cfg, err := config.Load()
			if err != nil {
				fmt.Printf("加载配置失败: %v\n", err)
				os.Exit(1)
			}

			successCount := 0
			duplicateCount := 0

			for _, server := range servers {
				// Basic validation
				if server.URL == "" {
					continue
				}

				group := server.Group
				if group == "" {
					group = "default"
					server.Group = "default"
				}

				// Check for duplicates before adding
				isDuplicate := false
				if groupServers, ok := cfg.Groups[group]; ok {
					for _, existing := range groupServers {
						if existing.URL == server.URL {
							isDuplicate = true
							break
						}
					}
				}

				config.AddServer(cfg, server)

				if isDuplicate {
					duplicateCount++
				} else {
					successCount++
				}
			}

			if err := config.Save(cfg); err != nil {
				fmt.Printf("保存配置失败: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("导入完成。\n成功导入: %d\n重复条目(已更新): %d\n", successCount, duplicateCount)
		},
	}

	return cmd
}

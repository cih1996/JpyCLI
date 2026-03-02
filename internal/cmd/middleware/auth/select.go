package auth

import (
	"fmt"
	"jpy-cli/pkg/config"
	"os"

	"github.com/spf13/cobra"
)

func NewSelectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "select [group]",
		Short: "选择活动分组",
		Long:  "选择后续操作的活动分组。如果未指定分组，则列出可用分组。",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			if err != nil {
				fmt.Printf("加载配置失败: %v\n", err)
				os.Exit(1)
			}

			if len(args) == 0 {
				fmt.Printf("当前活动分组: %s\n", cfg.ActiveGroup)
				fmt.Println("\n可用分组:")
				for group, servers := range cfg.Groups {
					prefix := "  "
					if group == cfg.ActiveGroup {
						prefix = "* "
					}
					fmt.Printf("%s%s (%d 台服务器)\n", prefix, group, len(servers))
				}
				return
			}

			targetGroup := args[0]
			if _, ok := cfg.Groups[targetGroup]; !ok {
				// Allow selecting non-existent group? Maybe, to start fresh.
				// But warning is good.
				fmt.Printf("警告: 分组 '%s' 尚不存在。添加服务器后将创建该分组。\n", targetGroup)
			}

			cfg.ActiveGroup = targetGroup
			if err := config.Save(cfg); err != nil {
				fmt.Printf("保存配置失败: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("活动分组已设置为: %s\n", targetGroup)
		},
	}

	return cmd
}

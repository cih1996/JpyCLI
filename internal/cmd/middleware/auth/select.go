package auth

import (
	"encoding/json"
	"fmt"
	"jpy-cli/pkg/config"

	"github.com/spf13/cobra"
)

func NewSelectCmd() *cobra.Command {
	var outputMode string

	cmd := &cobra.Command{
		Use:   "select [group]",
		Short: "选择活动分组",
		Long: `选择后续操作的活动分组。如果未指定分组，则列出可用分组。

输出模式:
  --output tui     默认输出
  --output plain   纯文本输出
  --output json    JSON 格式输出`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("加载配置失败: %v", err)
			}

			// 无参数：列出可用分组
			if len(args) == 0 {
				switch outputMode {
				case "json":
					return printSelectJSON(cfg)
				case "plain":
					return printSelectPlain(cfg)
				default:
					fmt.Printf("当前活动分组: %s\n", cfg.ActiveGroup)
					fmt.Println("\n可用分组:")
					for group, servers := range cfg.Groups {
						prefix := "  "
						if group == cfg.ActiveGroup {
							prefix = "* "
						}
						fmt.Printf("%s%s (%d 台服务器)\n", prefix, group, len(servers))
					}
					return nil
				}
			}

			// 指定分组：切换
			targetGroup := args[0]
			if _, ok := cfg.Groups[targetGroup]; !ok {
				if outputMode != "json" && outputMode != "plain" {
					fmt.Printf("警告: 分组 '%s' 尚不存在。添加服务器后将创建该分组。\n", targetGroup)
				}
			}

			cfg.ActiveGroup = targetGroup
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("保存配置失败: %v", err)
			}

			switch outputMode {
			case "json":
				result := map[string]interface{}{
					"active_group": targetGroup,
					"success":      true,
				}
				data, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(data))
			case "plain":
				fmt.Printf("OK\t%s\n", targetGroup)
			default:
				fmt.Printf("活动分组已设置为: %s\n", targetGroup)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputMode, "output", "o", "tui", "输出模式 (tui/plain/json)")
	return cmd
}

type selectGroupJSON struct {
	ActiveGroup string            `json:"active_group"`
	Groups      []selectGroupItem `json:"groups"`
}

type selectGroupItem struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
	Count  int    `json:"count"`
}

func printSelectJSON(cfg *config.Config) error {
	out := selectGroupJSON{
		ActiveGroup: cfg.ActiveGroup,
	}
	for g, servers := range cfg.Groups {
		out.Groups = append(out.Groups, selectGroupItem{
			Name:   g,
			Active: g == cfg.ActiveGroup,
			Count:  len(servers),
		})
	}
	data, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(data))
	return nil
}

func printSelectPlain(cfg *config.Config) error {
	fmt.Println("GROUP\tACTIVE\tCOUNT")
	for g, servers := range cfg.Groups {
		active := "false"
		if g == cfg.ActiveGroup {
			active = "true"
		}
		fmt.Printf("%s\t%s\t%d\n", g, active, len(servers))
	}
	return nil
}

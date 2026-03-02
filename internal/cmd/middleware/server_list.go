package middleware

import (
	"fmt"
	"jpy-cli/pkg/config"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func NewListCmd() *cobra.Command {
	var showHasFail bool
	var page int
	var pageSize int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出当前分组的服务器列表",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			activeGroup := cfg.ActiveGroup
			if activeGroup == "" {
				activeGroup = "default"
			}
			servers := cfg.Groups[activeGroup]

			var displayList []config.LocalServerConfig
			if showHasFail {
				for _, s := range servers {
					if s.Disabled {
						displayList = append(displayList, s)
					}
				}
			} else {
				displayList = servers
			}

			total := len(displayList)
			start := (page - 1) * pageSize
			end := start + pageSize
			if start >= total {
				fmt.Printf("页码超出范围 (总数: %d)\n", total)
				return nil
			}
			if end > total {
				end = total
			}

			fmt.Printf("当前分组: %s (总数: %d, 显示: %d-%d)\n", activeGroup, total, start+1, end)
			
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "INDEX\tURL\tUSERNAME\tDISABLED\tLAST ERROR")
			
			for i := start; i < end; i++ {
				s := displayList[i]
				disabledStr := ""
				if s.Disabled {
					disabledStr = "YES"
				}
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n", i+1, s.URL, s.Username, disabledStr, s.LastLoginError)
			}
			w.Flush()

			return nil
		},
	}

	cmd.Flags().BoolVar(&showHasFail, "has-fail", false, "只显示被软删除(连接失败)的服务器")
	cmd.Flags().IntVar(&page, "page", 1, "页码")
	cmd.Flags().IntVar(&pageSize, "size", 20, "每页数量")

	return cmd
}

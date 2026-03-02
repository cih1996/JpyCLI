package middleware

import (
	"encoding/json"
	"fmt"
	"jpy-cli/pkg/config"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func NewListCmd() *cobra.Command {
	var showHasFail bool
	var page int
	var pageSize int
	var output string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出当前分组的服务器列表",
		Long: `列出当前分组的服务器列表。

输出模式:
  --output tui     格式化表格（默认）
  --output plain   制表符分隔纯文本，适合 grep/awk
  --output json    JSON 格式，适合程序解析`,
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

			// JSON/plain 模式不分页，输出全部
			switch output {
			case "json":
				return printServerListJSON(activeGroup, displayList)
			case "plain":
				return printServerListPlain(activeGroup, displayList)
			}

			// TUI 模式（默认）：支持分页
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
	cmd.Flags().StringVarP(&output, "output", "o", "tui", "输出模式 (tui/plain/json)")

	return cmd
}

// --- Server List Output ---

type serverListJSON struct {
	Group   string           `json:"group"`
	Total   int              `json:"total"`
	Servers []serverItemJSON `json:"servers"`
}

type serverItemJSON struct {
	Index    int    `json:"index"`
	URL      string `json:"url"`
	Username string `json:"username"`
	Disabled bool   `json:"disabled"`
	Error    string `json:"error,omitempty"`
}

func cleanServerURL(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	return url
}

func printServerListJSON(group string, servers []config.LocalServerConfig) error {
	out := serverListJSON{
		Group:   group,
		Total:   len(servers),
		Servers: make([]serverItemJSON, len(servers)),
	}
	for i, s := range servers {
		out.Servers[i] = serverItemJSON{
			Index:    i + 1,
			URL:      cleanServerURL(s.URL),
			Username: s.Username,
			Disabled: s.Disabled,
			Error:    s.LastLoginError,
		}
	}
	data, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(data))
	return nil
}

func printServerListPlain(group string, servers []config.LocalServerConfig) error {
	fmt.Println("INDEX\tURL\tUSERNAME\tDISABLED\tERROR")
	for i, s := range servers {
		disabled := "false"
		if s.Disabled {
			disabled = "true"
		}
		fmt.Printf("%d\t%s\t%s\t%s\t%s\n", i+1, cleanServerURL(s.URL), s.Username, disabled, s.LastLoginError)
	}
	fmt.Fprintf(os.Stderr, "分组: %s, 总计: %d\n", group, len(servers))
	return nil
}

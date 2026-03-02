package middleware

import (
	"fmt"
	"jpy-cli/pkg/admin-middleware/api"
	"jpy-cli/pkg/admin-middleware/model"
	"jpy-cli/pkg/admin-middleware/service"
	"os"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

type listModel struct {
	table     table.Model
	client    *api.Client
	page      int
	total     int
	isLoading bool
	err       error
	width     int
	height    int
}

type authListMsg struct {
	result *model.AuthSearchResult
	page   int
}

type errMsg error

func initialModel(client *api.Client) listModel {
	columns := []table.Column{
		{Title: "ID", Width: 6},
		{Title: "名称", Width: 20},
		{Title: "序列号", Width: 35},
		{Title: "标题", Width: 20},
		{Title: "限制", Width: 8},
		{Title: "已用", Width: 8},
		{Title: "天数", Width: 8},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return listModel{
		table:     t,
		client:    client,
		page:      1,
		isLoading: true,
	}
}

func (m listModel) Init() tea.Cmd {
	return fetchList(m.client, m.page)
}

func fetchList(client *api.Client, page int) tea.Cmd {
	return func() tea.Msg {
		res, err := client.GetAuthList(page)
		if err != nil {
			return errMsg(err)
		}
		return authListMsg{result: res, page: page}
	}
}

func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "right", "n":
			if !m.isLoading && (m.page*20 < m.total) { // Assuming limit is 20
				m.page++
				m.isLoading = true
				return m, fetchList(m.client, m.page)
			}
		case "left", "p":
			if !m.isLoading && m.page > 1 {
				m.page--
				m.isLoading = true
				return m, fetchList(m.client, m.page)
			}
		}
	case authListMsg:
		m.isLoading = false
		m.total = msg.result.Data.Total
		m.page = msg.page // Confirm page

		rows := []table.Row{}
		for _, item := range msg.result.Data.DataList {
			rows = append(rows, table.Row{
				fmt.Sprintf("%d", item.ID),
				item.Name,
				item.SerialNumber,
				item.Title,
				fmt.Sprintf("%d", item.Limit),
				fmt.Sprintf("%d", item.Used),
				fmt.Sprintf("%d", item.Day),
			})
		}
		m.table.SetRows(rows)
		// Update footer or title with pagination info
	case errMsg:
		m.isLoading = false
		m.err = msg
		return m, nil // Don't quit, show error
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetWidth(msg.Width - 4)
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m listModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("错误: %v\n\n按 q 退出", m.err)
	}

	header := fmt.Sprintf("设备授权管理列表 - 第 %d 页 (共 %d 条)", m.page, m.total)
	if m.isLoading {
		header += " [加载中...]"
	}

	return baseStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			header,
			m.table.View(),
			"操作: ←/p 上一页 • →/n 下一页 • q/Esc 退出",
		),
	) + "\n"
}

func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "查看授权码列表 (TUI)",
		Run: func(cmd *cobra.Command, args []string) {
			// 1. Ensure Login
			adminCfg, err := service.EnsureLoggedIn()
			if err != nil {
				fmt.Printf("需要登录: %v\n", err)
				return
			}

			client := api.NewClient(adminCfg.Token)

			p := tea.NewProgram(initialModel(client))
			if _, err := p.Run(); err != nil {
				fmt.Printf("TUI 运行错误: %v\n", err)
				os.Exit(1)
			}
		},
	}
	return cmd
}

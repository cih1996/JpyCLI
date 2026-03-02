package auth

import (
	"fmt"
	httpclient "jpy-cli/pkg/client/http"
	"jpy-cli/pkg/config"
	"os"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	detailsGroup string
	concurrency  int
	plainOutput  bool
	sem          chan struct{}
)

func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出服务器",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.Load()
			if err != nil {
				fmt.Printf("无法加载配置: %v\n", err)
				return
			}

			// Initialize semaphore
			if !cmd.Flags().Changed("concurrency") && config.GlobalSettings.MaxConcurrency > 0 {
				concurrency = config.GlobalSettings.MaxConcurrency
			}
			if concurrency < 1 {
				concurrency = 1
			}
			sem = make(chan struct{}, concurrency)

			if detailsGroup != "" {
				showGroupDetails(cfg, detailsGroup, concurrency)
			} else if plainOutput {
				showPlainList(cfg)
			} else {
				runTUI(cfg)
			}
		},
	}

	cmd.Flags().StringVarP(&detailsGroup, "details", "d", "", "显示特定分组的详情")
	cmd.Flags().IntVarP(&concurrency, "concurrency", "c", 5, "服务器检查的并发限制")
	cmd.Flags().BoolVar(&plainOutput, "plain", false, "纯文本输出（适用于非交互式环境）")

	return cmd
}

// --- Plain Text Mode (非交互式输出) ---

func showPlainList(cfg *config.Config) {
	activeGroup := cfg.ActiveGroup
	if activeGroup == "" {
		activeGroup = "default"
	}

	fmt.Printf("分组列表:\n")
	for g, servers := range cfg.Groups {
		active := ""
		if g == activeGroup {
			active = " (当前)"
		}
		fmt.Printf("  %s%s - %d 台服务器\n", g, active, len(servers))
	}

	fmt.Printf("\n当前分组 [%s] 服务器:\n", activeGroup)
	servers := config.GetGroupServers(cfg, activeGroup)
	if len(servers) == 0 {
		fmt.Println("  (空)")
		return
	}

	fmt.Printf("  %-4s %-30s %-15s %-10s\n", "序号", "URL", "用户名", "状态")
	fmt.Printf("  %-4s %-30s %-15s %-10s\n", "----", "------------------------------", "---------------", "----------")
	for i, s := range servers {
		status := "正常"
		if s.Disabled {
			status = "已移除"
		} else if s.LastLoginError != "" {
			status = "异常"
		}
		fmt.Printf("  %-4d %-30s %-15s %-10s\n", i+1, s.URL, s.Username, status)
	}
}

// --- Non-Interactive Mode ---

func showGroupDetails(cfg *config.Config, groupName string, concurrency int) {
	servers := config.GetGroupServers(cfg, groupName)

	if len(servers) == 0 {
		fmt.Printf("在分组 '%s' 中未找到服务器\n", groupName)
		return
	}

	fmt.Printf("正在检查分组 '%s' 中 %d 台服务器的状态 (并发: %d)...\n", groupName, len(servers), concurrency)

	results := make(chan config.LocalServerConfig, len(servers))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for _, server := range servers {
		wg.Add(1)
		go func(s config.LocalServerConfig) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Perform check/login
			updatedServer := checkServerStatus(s)
			results <- updatedServer
		}(server)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results and update config
	// We need to keep other groups' servers too
	// Map existing servers by URL for easy replacement
	serverMap := make(map[string]config.LocalServerConfig)
	for _, s := range cfg.Servers {
		serverMap[s.URL] = s
	}

	// Output table header
	fmt.Printf("%-30s %-15s %-20s %-30s\n", "URL", "用户名", "状态", "最后错误")
	fmt.Println("----------------------------------------------------------------------------------------------------")

	for s := range results {
		status := "正常"
		if s.LastLoginError != "" {
			status = "失败"
		}
		fmt.Printf("%-30s %-15s %-20s %-30s\n", s.URL, s.Username, status, truncate(s.LastLoginError, 30))
		serverMap[s.URL] = s
	}

	// Reconstruct config servers list
	var newServers []config.LocalServerConfig
	for _, s := range cfg.Servers {
		if updated, ok := serverMap[s.URL]; ok {
			newServers = append(newServers, updated)
		} else {
			newServers = append(newServers, s)
		}
	}
	cfg.Servers = newServers
	config.Save(cfg)
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}

func checkServerStatus(server config.LocalServerConfig) config.LocalServerConfig {
	client := httpclient.NewClient(server.URL, server.Token)
	// Try to login if no token or just to verify
	// User wants "login status". The best way is to try login.
	// If we already have a token, we could try to use it (e.g. GetLicense), but login is safer to refresh.
	// Let's just try login.

	_, err := client.Login(server.Username, server.Password)
	server.LastLoginTime = time.Now().Format(time.RFC3339)
	if err != nil {
		server.LastLoginError = err.Error()
	} else {
		server.LastLoginError = ""
		server.Token = client.Token // Update token
	}
	return server
}

// --- TUI Mode ---

type viewState int

const (
	viewGroups viewState = iota
	viewServers
)

type item struct {
	title string
	desc  string
}

type modelTUI struct {
	cfg           *config.Config
	state         viewState
	groups        []string
	servers       []config.LocalServerConfig // Current group servers
	cursor        int
	selectedGroup string

	// Server list status
	serverStatus map[string]string // URL -> Status
	checking     bool
	spinner      spinner.Model
}

func runTUI(cfg *config.Config) {
	// Extract groups
	var groups []string
	for g := range cfg.Groups {
		groups = append(groups, g)
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := modelTUI{
		cfg:          cfg,
		state:        viewGroups,
		groups:       groups,
		serverStatus: make(map[string]string),
		spinner:      s,
	}

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}

func (m modelTUI) Init() tea.Cmd {
	return m.spinner.Tick
}

type statusMsg struct {
	url    string
	status string
	err    string
}

func checkServerCmd(server config.LocalServerConfig) tea.Cmd {
	return func() tea.Msg {
		sem <- struct{}{}
		defer func() { <-sem }()

		updated := checkServerStatus(server)
		status := "在线"
		if updated.LastLoginError != "" {
			status = "失败: " + updated.LastLoginError
		}
		return statusMsg{url: server.URL, status: status, err: updated.LastLoginError}
	}
}

func (m modelTUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			limit := 0
			if m.state == viewGroups {
				limit = len(m.groups) - 1
			} else {
				limit = len(m.servers) - 1
			}
			if m.cursor < limit {
				m.cursor++
			}
		case "enter":
			if m.state == viewGroups {
				if len(m.groups) == 0 {
					return m, nil
				}
				m.selectedGroup = m.groups[m.cursor]
				m.state = viewServers
				m.cursor = 0
				// Filter servers
				m.servers = config.GetGroupServers(m.cfg, m.selectedGroup)

				// Trigger checks
				m.checking = true
				var cmds []tea.Cmd
				for _, s := range m.servers {
					m.serverStatus[s.URL] = "检查中..."
					cmds = append(cmds, checkServerCmd(s))
				}
				return m, tea.Batch(cmds...)
			} else {
				// Detail view for server? Or just toggle?
				// For now, nothing special on Enter in server list
			}
		case "esc":
			if m.state == viewServers {
				m.state = viewGroups
				m.cursor = 0
				m.checking = false
				return m, nil
			}
		}

	case statusMsg:
		m.serverStatus[msg.url] = msg.status
		// Update config in memory
		if servers, ok := m.cfg.Groups[m.selectedGroup]; ok {
			for i, s := range servers {
				if s.URL == msg.url {
					servers[i].LastLoginError = msg.err
					servers[i].LastLoginTime = time.Now().Format(time.RFC3339)
					break
				}
			}
			m.cfg.Groups[m.selectedGroup] = servers
		}
		// Save config? Maybe not in TUI to avoid lag, or save on exit?
		// User asked to record it.
		config.Save(m.cfg)
		// but here we process one msg at a time in Update loop, so it's safe.
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m modelTUI) View() string {
	s := ""

	if m.state == viewGroups {
		s += lipgloss.NewStyle().Bold(true).Render("选择一个分组:") + "\n\n"
		for i, g := range m.groups {
			cursor := "  "
			if m.cursor == i {
				cursor = "> "
			}
			s += fmt.Sprintf("%s%s\n", cursor, g)
		}
		s += "\n(按 Enter 查看服务器, q 退出)"
	} else {
		s += lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("分组中的服务器: %s", m.selectedGroup)) + "\n"
		if m.checking {
			s += fmt.Sprintf("%s 正在检查状态...\n", m.spinner.View())
		}
		s += "\n"

		// Header
		s += fmt.Sprintf("  %-30s %-20s\n", "URL", "状态")
		s += "  " + lipgloss.NewStyle().Strikethrough(true).Render("--------------------------------------------------") + "\n"

		for i, srv := range m.servers {
			cursor := "  "
			if m.cursor == i {
				cursor = "> "
			}
			status := m.serverStatus[srv.URL]
			if status == "" {
				status = "未知"
			}

			// Color status
			statusStyle := lipgloss.NewStyle()
			if status == "在线" {
				statusStyle = statusStyle.Foreground(lipgloss.Color("42")) // Green
			} else if status == "检查中..." {
				statusStyle = statusStyle.Foreground(lipgloss.Color("205")) // Pink
			} else {
				statusStyle = statusStyle.Foreground(lipgloss.Color("196")) // Red
			}

			s += fmt.Sprintf("%s%-30s %s\n", cursor, srv.URL, statusStyle.Render(status))
		}
		s += "\n(按 Esc 返回, q 退出)"
	}

	return s
}

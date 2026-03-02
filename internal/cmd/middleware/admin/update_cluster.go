package admin

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"jpy-cli/pkg/admin-middleware/api"
	adminModel "jpy-cli/pkg/admin-middleware/model"
	"jpy-cli/pkg/admin-middleware/service"
	httpclient "jpy-cli/pkg/client/http"
	"jpy-cli/pkg/config"
	"jpy-cli/pkg/logger"
	"jpy-cli/pkg/middleware/device/selector"
	deviceModel "jpy-cli/pkg/middleware/model"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

type UpdateClusterFlags struct {
	Group         string
	ServerPattern string
	Authorized    string
	Force         bool
}

func NewUpdateClusterCmd() *cobra.Command {
	var opts UpdateClusterFlags

	cmd := &cobra.Command{
		Use:   "update-cluster [new_address]",
		Short: "批量更新中间件服务器的集控平台地址",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetAddr := args[0]
			if targetAddr == "" {
				return fmt.Errorf("集控平台地址不能为空")
			}

			// 1. Load Config
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			targetGroup := opts.Group
			if targetGroup == "" {
				targetGroup = cfg.ActiveGroup
			}
			if targetGroup == "" {
				targetGroup = "default"
			}

			fmt.Printf("当前分组: %s\n", targetGroup)
			fmt.Printf("目标集控地址: %s\n", targetAddr)

			// 2. Scan and Filter Servers
			fmt.Println("正在扫描服务器状态...")
			servers := config.GetGroupServers(cfg, targetGroup)
			var candidates []candidateServer
			var mu sync.Mutex
			var wg sync.WaitGroup

			sem := make(chan struct{}, 10) // Concurrency 10

			for _, s := range servers {
				if s.Disabled {
					continue
				}
				if !selector.MatchServerPattern(s.URL, opts.ServerPattern) {
					continue
				}

				wg.Add(1)
				go func(server config.LocalServerConfig) {
					defer wg.Done()
					sem <- struct{}{}
					defer func() { <-sem }()

					client := httpclient.NewClient(server.URL, server.Token)
					lic, err := client.GetLicense()
					if err != nil {
						// Try relogin once
						newToken, loginErr := client.Login(server.Username, server.Password)
						if loginErr == nil {
							server.Token = newToken
							config.UpdateServer(cfg, server)
							client.Token = newToken
							lic, err = client.GetLicense()
						}
					}

					if err != nil {
						return // Skip offline servers
					}

					// Filter by Authorized flag
					isAuthorized := lic.Status == 1 || lic.S
					if opts.Authorized == "true" && !isAuthorized {
						return
					}
					if opts.Authorized == "false" && isAuthorized {
						return
					}

					// Check if update is needed
					if lic.C == targetAddr && !opts.Force {
						return // Already matches
					}

					// Check SN validity for admin search
					if lic.Sn == "" {
						return
					}

					mu.Lock()
					candidates = append(candidates, candidateServer{
						Server:  server,
						License: lic,
					})
					mu.Unlock()
				}(s)
			}
			wg.Wait()

			if len(candidates) == 0 {
				fmt.Println("没有发现需要更新的服务器。")
				return nil
			}

			// Sort candidates by URL for better UX
			sort.Slice(candidates, func(i, j int) bool {
				return candidates[i].Server.URL < candidates[j].Server.URL
			})

			fmt.Printf("发现 %d 台服务器需要更新。\n", len(candidates))

			// 3. Admin Login
			adminCfg, err := service.EnsureLoggedIn()
			if err != nil {
				return err
			}
			adminClient := api.NewClient(adminCfg.Token)

			// 4. Start TUI
			p := tea.NewProgram(newModel(candidates, targetAddr, adminClient, opts.Force))
			if _, err := p.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&opts.Group, "group", "", "指定服务器分组")
	cmd.Flags().StringVar(&opts.ServerPattern, "server", "", "筛选服务器地址 (支持正则/模糊匹配，多条件用|分隔)")
	cmd.Flags().StringVar(&opts.Authorized, "authorized", "", "筛选授权状态 (true/false)")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "强制更新（即使地址一致也重新提交）")

	return cmd
}

type candidateServer struct {
	Server  config.LocalServerConfig
	License *deviceModel.LicenseData
}

type failureDetail struct {
	Address string
	Reason  string
}

// TUI Model
type updateClusterModel struct {
	candidates   []candidateServer
	targetAddr   string
	adminClient  *api.Client
	force        bool
	index        int
	logs         []string
	progress     progress.Model
	viewport     viewport.Model
	width        int
	height       int
	done         bool
	successCount int
	failureCount int
	failures     []failureDetail
}

type logMsg string
type progressMsg float64
type doneMsg struct{}
type stepResultMsg struct {
	logs         []string
	progress     float64
	nextIndex    int
	step         int
	success      bool
	failedServer string
	errorMsg     string
}
type updateNextMsg struct {
	index int
	step  int
}

func newModel(candidates []candidateServer, targetAddr string, adminClient *api.Client, force bool) updateClusterModel {
	p := progress.New(progress.WithDefaultGradient())
	return updateClusterModel{
		candidates:  candidates,
		targetAddr:  targetAddr,
		adminClient: adminClient,
		force:       force,
		progress:    p,
		logs:        make([]string, 0),
	}
}

func (m updateClusterModel) Init() tea.Cmd {
	// Start with single worker to avoid concurrency issues with middleware reauthorization
	return processNext(m.candidates, m.targetAddr, m.adminClient, 0, 1, m.force)
}

func (m updateClusterModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = msg.Width - 4
		m.viewport = viewport.New(msg.Width, msg.Height-4) // Reserve space for progress and title
		m.viewport.SetContent(strings.Join(m.logs, "\n"))
		return m, nil
	case logMsg:
		m.logs = append(m.logs, string(msg))
		if len(m.logs) > 100 { // Keep buffer reasonable
			m.logs = m.logs[len(m.logs)-100:]
		}
		displayLogs := m.logs
		if len(displayLogs) > 10 {
			displayLogs = displayLogs[len(displayLogs)-10:]
		}
		m.viewport.SetContent(strings.Join(displayLogs, "\n"))
		m.viewport.GotoBottom()
		return m, nil
	case progressMsg:
		cmd := m.progress.SetPercent(float64(msg))
		return m, cmd
	case doneMsg:
		m.done = true
		return m, tea.Quit
	case updateNextMsg:
		return m, processNext(m.candidates, m.targetAddr, m.adminClient, msg.index, msg.step, m.force)
	case stepResultMsg:
		if msg.success {
			m.successCount++
		} else {
			m.failureCount++
			m.failures = append(m.failures, failureDetail{Address: msg.failedServer, Reason: msg.errorMsg})
		}

		m.index++ // Track overall progress

		cmds := []tea.Cmd{}
		for _, l := range msg.logs {
			// Capture loop variable
			line := l
			cmds = append(cmds, func() tea.Msg { return logMsg(line) })
		}
		cmds = append(cmds, m.progress.SetPercent(float64(m.index)/float64(len(m.candidates))))

		// Continue with next item in stride
		cmds = append(cmds, func() tea.Msg { return updateNextMsg{index: msg.nextIndex, step: msg.step} })
		return m, tea.Batch(cmds...)
	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
	}
	return m, nil
}

func (m updateClusterModel) View() string {
	if m.done {
		summary := fmt.Sprintf("\n更新完成！共处理 %d 台服务器。\n", len(m.candidates))
		summary += fmt.Sprintf("✅ 成功: %d\n", m.successCount)
		summary += fmt.Sprintf("❌ 失败: %d\n", m.failureCount)

		if len(m.failures) > 0 {
			summary += "\n失败详情:\n"
			for _, f := range m.failures {
				summary += fmt.Sprintf("- %s: %s\n", f.Address, f.Reason)
			}
		}
		return summary
	}

	pad := strings.Repeat(" ", padding)
	return "\n" +
		pad + m.progress.View() + "\n\n" +
		m.viewport.View() + "\n\n" +
		pad + helpStyle("按 'q' 退出")
}

var (
	padding   = 2
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render
)

func processNext(candidates []candidateServer, targetAddr string, adminClient *api.Client, index int, step int, force bool) tea.Cmd {
	return func() tea.Msg {
		if index >= len(candidates) {
			return doneMsg{}
		}

		c := candidates[index]
		server := c.Server
		sn := c.License.Sn

		var logs []string
		// Compact log: single line per server
		// logs = append(logs, fmt.Sprintf("正在处理: %s (SN: %s)", server.URL, sn))

		var err error
		var actionStatus string // "updated", "skipped", "failed"
		maxRetries := 3

		for attempt := 1; attempt <= maxRetries; attempt++ {
			err = func() error {
				// Always check Admin Auth first because candidates are here implies lic.C != targetAddr or Force is true
				logger.Info(fmt.Sprintf("[AUDIT] Checking Admin Auth for SN: %s (Server: %s)", sn, server.URL))

				// 1. Get Auth Info from Admin
				authItem, err := adminClient.GetAuthBySN(sn)
				if err != nil {
					return fmt.Errorf("获取授权信息失败: %v", err)
				}

				// Optimization: Check if Admin Backend is already correct
				if authItem.MgtCenter == targetAddr && !force {
					logger.Info(fmt.Sprintf("[AUDIT] SN: %s - Admin MgtCenter already matches target (%s). Skipping Admin Update.", sn, targetAddr))
					// We skip Admin Update, but MUST continue to Reauthorize Middleware because candidates here have mismatched C.
					actionStatus = "skipped"
				} else {
					// 2. Update Auth Info
					payload := adminModel.AuthCodePayload{
						ID:           authItem.ID,
						Supervise:    authItem.Supervise,
						Type:         authItem.Type,
						Name:         authItem.Name,
						SerialNumber: authItem.SerialNumber,
						Title:        authItem.Title,
						MgtCenter:    targetAddr, // Update target
						Limit:        authItem.Limit,
						Day:          authItem.Day,
						Desc:         authItem.Desc,
					}

					if force && authItem.MgtCenter == targetAddr {
						logger.Info(fmt.Sprintf("[AUDIT] SN: %s - Admin MgtCenter matches target (%s) but FORCE is enabled. Updating anyway.", sn, targetAddr))
					}

					logger.Info(fmt.Sprintf("[AUDIT] SN: %s - Updating Admin MgtCenter to %s (Old: %s)", sn, targetAddr, authItem.MgtCenter))

					if err := adminClient.UpdateAuth(payload); err != nil {
						return fmt.Errorf("更新授权失败: %v", err)
					}
					actionStatus = "updated"
				}

				// 3. Reauthorize Middleware (ALWAYS perform this step if we are here)
				// We are here because lic.C != targetAddr or Force is true, so we must force the middleware to refresh.
				logger.Info(fmt.Sprintf("[AUDIT] SN: %s - Reauthorizing Middleware at %s", sn, server.URL))
				middlewareClient := httpclient.NewClient(server.URL, server.Token)
				if err := middlewareClient.Reauthorize(sn); err != nil {
					return fmt.Errorf("中间件重新授权失败: %v", err)
				}

				return nil
			}()

			if err == nil {
				break
			}

			if attempt < maxRetries {
				// logs = append(logs, fmt.Sprintf("⚠️ %s 操作失败，正在重试 (%d/%d): %v", server.URL, attempt, maxRetries, err))
				time.Sleep(2 * time.Second)
			}
		}

		success := err == nil
		var errorMsg string
		if err != nil {
			errorMsg = err.Error()
			logs = append(logs, fmt.Sprintf("❌ %s 失败: %v", server.URL, err))
			logger.Info(fmt.Sprintf("[AUDIT] SN: %s - Update failed: %v", sn, err))
		} else {
			if actionStatus == "skipped" {
				logs = append(logs, fmt.Sprintf("✅ %s 重新授权成功 (后台地址已一致)", server.URL))
			} else {
				logs = append(logs, fmt.Sprintf("✅ %s 重新授权成功 (后台地址已更新)", server.URL))
			}
		}

		// Calculate progress - no longer needed here as tracked in model
		prog := 0.0

		return stepResultMsg{
			logs:         logs,
			progress:     prog,
			nextIndex:    index + step,
			step:         step,
			success:      success,
			failedServer: server.URL,
			errorMsg:     errorMsg,
		}
	}
}

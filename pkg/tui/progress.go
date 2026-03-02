package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render
	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Render
)

type ProgressMsg struct {
	Result interface{}
}

type ProgressFinishMsg struct{}

type ProgressModel struct {
	progress   progress.Model
	total      int
	current    int
	results    []interface{}
	sub        chan interface{}
	statusFunc func(interface{}) string
	lastStatus string
}

func NewProgressModel(total int, sub chan interface{}, statusFunc func(interface{}) string) ProgressModel {
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)
	return ProgressModel{
		progress:   p,
		total:      total,
		sub:        sub,
		statusFunc: statusFunc,
		results:    make([]interface{}, 0, total),
	}
}

func (m ProgressModel) Init() tea.Cmd {
	return waitForUpdate(m.sub)
}

func waitForUpdate(sub chan interface{}) tea.Cmd {
	return func() tea.Msg {
		data, ok := <-sub
		if !ok {
			return ProgressFinishMsg{}
		}
		return ProgressMsg{Result: data}
	}
}

func (m ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.progress.Width = msg.Width - 10
		if m.progress.Width > 80 {
			m.progress.Width = 80
		}
		return m, nil

	case ProgressMsg:
		m.current++
		m.results = append(m.results, msg.Result)
		if m.statusFunc != nil {
			m.lastStatus = m.statusFunc(msg.Result)
		}

		cmd := m.progress.SetPercent(float64(m.current) / float64(m.total))
		return m, tea.Batch(cmd, waitForUpdate(m.sub))

	case ProgressFinishMsg:
		return m, tea.Quit

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd
	}

	return m, nil
}

func (m ProgressModel) View() string {
	pct := float64(m.current) / float64(m.total) * 100
	return fmt.Sprintf(
		"\n%s\n\n%s %d/%d (%.0f%%)\n%s\n",
		m.progress.View(),
		helpStyle("正在处理..."),
		m.current, m.total, pct,
		statusStyle(m.lastStatus),
	)
}

func (m ProgressModel) GetResults() []interface{} {
	return m.results
}

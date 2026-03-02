package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true).MarginBottom(1)
	itemStyle     = lipgloss.NewStyle().PaddingLeft(2)
	headerStyle   = lipgloss.NewStyle().PaddingLeft(4).Foreground(lipgloss.Color("240"))
	selectedStyle = lipgloss.NewStyle().PaddingLeft(0).Foreground(lipgloss.Color("205")).Bold(true)
)

type Option struct {
	Label string
	Value string
}

type SelectionModel struct {
	Title    string
	Header   string
	Options  []Option
	Cursor   int
	Selected *Option
	Quitting bool
}

func (m SelectionModel) Init() tea.Cmd {
	return nil
}

func (m SelectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.Quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "down", "j":
			if m.Cursor < len(m.Options)-1 {
				m.Cursor++
			}
		case "enter", " ":
			m.Selected = &m.Options[m.Cursor]
			m.Quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m SelectionModel) View() string {
	if m.Selected != nil {
		return ""
	}
	if m.Quitting {
		return "操作已取消\n"
	}

	s := titleStyle.Render(m.Title) + "\n"

	if m.Header != "" {
		s += headerStyle.Render(m.Header) + "\n"
	}

	for i, option := range m.Options {
		cursor := "  "
		style := itemStyle
		if m.Cursor == i {
			cursor = "> "
			style = selectedStyle
		}
		s += style.Render(fmt.Sprintf("%s%s", cursor, option.Label)) + "\n"
	}

	s += "\n(使用 ↑/↓ 选择，Enter 确认)\n"
	return s
}

// SelectOption prompts the user to select one option from the list
func SelectOption(title string, header string, options []Option) (string, error) {
	m := SelectionModel{
		Title:   title,
		Header:  header,
		Options: options,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	final := finalModel.(SelectionModel)
	if final.Selected == nil {
		return "", fmt.Errorf("未选择任何选项")
	}

	return final.Selected.Value, nil
}

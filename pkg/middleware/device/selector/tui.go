package selector

import (
	"fmt"
	"jpy-cli/pkg/middleware/model"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type InteractiveSelectionModel struct {
	table     table.Model
	devices   []model.DeviceInfo
	selected  map[int]bool // Key is index in devices slice
	quitting  bool
	confirmed bool
}

func (m InteractiveSelectionModel) Init() tea.Cmd { return nil }

func (m InteractiveSelectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			m.confirmed = true
			m.quitting = true
			return m, tea.Quit
		case " ":
			idx := m.table.Cursor()
			if m.selected[idx] {
				delete(m.selected, idx)
			} else {
				m.selected[idx] = true
			}
			// Update table rows to reflect selection
			m.updateRows()
			return m, nil
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *InteractiveSelectionModel) updateRows() {
	rows := []table.Row{}
	for i, d := range m.devices {
		mark := "[ ]"
		if m.selected[i] {
			mark = "[x]"
		}

		online := "离线"
		if d.IsOnline {
			online = "在线"
		}

		biz := "-"
		if d.BizOnline {
			biz = "是"
		}

		adb := "关闭"
		if d.ADBEnabled {
			adb = "开启"
		}

		usb := "OTG"
		if d.USBMode {
			usb = "USB"
		}

		rows = append(rows, table.Row{
			mark,
			cleanURL(d.ServerURL),
			fmt.Sprintf("%d", d.Seat),
			d.UUID,
			d.IP,
			d.Model,
			online,
			biz,
			adb,
			usb,
		})
	}
	m.table.SetRows(rows)
}

func (m *InteractiveSelectionModel) sortDevices() {
	sort.Slice(m.devices, func(i, j int) bool {
		if m.devices[i].ServerURL == m.devices[j].ServerURL {
			return m.devices[i].Seat < m.devices[j].Seat
		}
		return m.devices[i].ServerURL < m.devices[j].ServerURL
	})
}

func (m InteractiveSelectionModel) View() string {
	if m.quitting {
		return ""
	}
	return baseStyle.Render(m.table.View()) + "\n[空格] 切换  [Enter] 确认  [q] 退出\n"
}

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

func RunInteractiveSelection(devices []model.DeviceInfo) ([]model.DeviceInfo, error) {
	columns := []table.Column{
		{Title: "选", Width: 4},
		{Title: "服务器", Width: 16},
		{Title: "盘位", Width: 4},
		{Title: "UUID", Width: 30},
		{Title: "IP", Width: 15},
		{Title: "型号", Width: 12},
		{Title: "在线", Width: 8},
		{Title: "业务", Width: 4},
		{Title: "ADB", Width: 4},
		{Title: "USB", Width: 4},
	}

	m := InteractiveSelectionModel{
		devices:  devices,
		selected: make(map[int]bool),
	}
	m.sortDevices()

	// Create custom keymap to unbind Space from PageDown
	km := table.DefaultKeyMap()
	km.PageDown.SetKeys()
	km.PageUp.SetKeys()
	km.HalfPageDown.SetKeys()
	km.HalfPageUp.SetKeys()

	// Initialize table with rows
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
		table.WithKeyMap(km),
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

	m.table = t
	m.updateRows()

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	final := finalModel.(InteractiveSelectionModel)
	if !final.confirmed {
		return nil, fmt.Errorf("选择已取消")
	}

	var selectedDevices []model.DeviceInfo
	for i, d := range final.devices {
		if final.selected[i] {
			selectedDevices = append(selectedDevices, d)
		}
	}

	// If nothing selected but confirmed, check if only one device is available or default to cursor?
	// For safety, require explicit selection or fallback to "current cursor" if nothing checked?
	// Let's implement: if nothing checked, select the one under cursor.
	if len(selectedDevices) == 0 {
		idx := final.table.Cursor()
		if idx >= 0 && idx < len(final.devices) {
			selectedDevices = append(selectedDevices, final.devices[idx])
		}
	}

	if len(selectedDevices) == 0 {
		return nil, fmt.Errorf("未选择任何设备")
	}

	return selectedDevices, nil
}

func cleanURL(url string) string {
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "https://")
	return url
}

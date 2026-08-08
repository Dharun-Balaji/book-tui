package settings

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dharuncs/novel/internal/storage"
	"github.com/dharuncs/novel/internal/tui/styles"
)

type SaveSettingsMsg struct {
	Settings storage.UserSettings
}

type Model struct {
	settings storage.UserSettings
	theme    styles.Theme
	cursor   int
	width    int
	height   int
}

func New() Model {
	return Model{
		settings: storage.UserSettings{
			Theme:         "dark",
			LineWidth:     80,
			AutoSaveEvery: 5,
		},
		theme: styles.DarkTheme,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) SetSettings(s storage.UserSettings) Model {
	m.settings = s
	m.theme = styles.GetTheme(s.Theme)
	return m
}

func (m Model) SetSize(width, height int) Model {
	m.width = width
	m.height = height
	return m
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < 2 {
				m.cursor++
			}
		case "enter", "space":
			switch m.cursor {
			case 0: // Toggle Theme
				if m.settings.Theme == "dark" {
					m.settings.Theme = "light"
				} else {
					m.settings.Theme = "dark"
				}
				m.theme = styles.GetTheme(m.settings.Theme)
			case 1: // Cycle Line Width
				switch m.settings.LineWidth {
				case 60:
					m.settings.LineWidth = 80
				case 80:
					m.settings.LineWidth = 100
				default:
					m.settings.LineWidth = 60
				}
			case 2: // Cycle Auto-Save Threshold
				switch m.settings.AutoSaveEvery {
				case 3:
					m.settings.AutoSaveEvery = 5
				case 5:
					m.settings.AutoSaveEvery = 10
				default:
					m.settings.AutoSaveEvery = 3
				}
			}
			// Emit SaveSettingsMsg on any change
			s := m.settings
			return m, func() tea.Msg {
				return SaveSettingsMsg{Settings: s}
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	header := m.theme.Header.Render("Settings & Preferences")

	optTheme := fmt.Sprintf("Theme: [%s]", m.settings.Theme)
	optWidth := fmt.Sprintf("Line Width: [%d]", m.settings.LineWidth)
	optSave := fmt.Sprintf("Auto-Save Every: [%d paragraphs]", m.settings.AutoSaveEvery)

	opts := []string{optTheme, optWidth, optSave}
	var renderedOpts []string

	for i, opt := range opts {
		if i == m.cursor {
			renderedOpts = append(renderedOpts, m.theme.Selected.Render("> "+opt))
		} else {
			renderedOpts = append(renderedOpts, lipgloss.NewStyle().Foreground(m.theme.Foreground).Render("  "+opt))
		}
	}

	footer := m.theme.Footer.Render("\n↑/k: up • ↓/j: down • enter/space: toggle • q/esc: back")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		lipgloss.JoinVertical(lipgloss.Left, renderedOpts...),
		footer,
	)

	return content
}

package styles

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Name       string
	Background lipgloss.Color
	Foreground lipgloss.Color
	Header     lipgloss.Style
	Footer     lipgloss.Style
	Title      lipgloss.Style
	Subtitle   lipgloss.Style
	Accent     lipgloss.Style
	Selected   lipgloss.Style
	Border     lipgloss.Style
	Status     lipgloss.Style
	Muted      lipgloss.Style
}

var DarkTheme = Theme{
	Name:       "dark",
	Background: lipgloss.Color("#1a1b26"),
	Foreground: lipgloss.Color("#c0caf5"),
	Header:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#bb9af7")),
	Footer:     lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89")),
	Title:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7aa2f7")),
	Subtitle:   lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89")),
	Accent:     lipgloss.NewStyle().Foreground(lipgloss.Color("#bb9af7")),
	Selected:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7aa2f7")).Underline(true),
	Border:     lipgloss.NewStyle().BorderForeground(lipgloss.Color("#3b4261")),
	Status:     lipgloss.NewStyle().Foreground(lipgloss.Color("#e0af68")),
	Muted:      lipgloss.NewStyle().Foreground(lipgloss.Color("#565f89")),
}

var LightTheme = Theme{
	Name:       "light",
	Background: lipgloss.Color("#ffffff"),
	Foreground: lipgloss.Color("#24292e"),
	Header:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#6f42c1")),
	Footer:     lipgloss.NewStyle().Foreground(lipgloss.Color("#6a737d")),
	Title:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0366d6")),
	Subtitle:   lipgloss.NewStyle().Foreground(lipgloss.Color("#6a737d")),
	Accent:     lipgloss.NewStyle().Foreground(lipgloss.Color("#6f42c1")),
	Selected:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#0366d6")).Underline(true),
	Border:     lipgloss.NewStyle().BorderForeground(lipgloss.Color("#e1e4e8")),
	Status:     lipgloss.NewStyle().Foreground(lipgloss.Color("#b08800")),
	Muted:      lipgloss.NewStyle().Foreground(lipgloss.Color("#6a737d")),
}

func GetTheme(name string) Theme {
	if name == "light" {
		return LightTheme
	}
	return DarkTheme
}

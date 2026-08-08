package library

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dharuncs/novel/internal/core"
)

type item struct {
	novel core.Novel
}

func (i item) FilterValue() string { return i.novel.Title + " " + i.novel.Author }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 2 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	selectedTitleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170"))

	if index == m.Index() {
		fmt.Fprintf(w, "> %s\n  %s", selectedTitleStyle.Render(i.novel.Title), descStyle.Render(fmt.Sprintf("%s • %d ch", i.novel.Author, i.novel.TotalChapters)))
	} else {
		fmt.Fprintf(w, "  %s\n  %s", titleStyle.Render(i.novel.Title), descStyle.Render(fmt.Sprintf("%s • %d ch", i.novel.Author, i.novel.TotalChapters)))
	}
}

type Model struct {
	list   list.Model
	novels []core.Novel
	width  int
	height int
}

func New() Model {
	l := list.New([]list.Item{}, itemDelegate{}, 0, 0)
	l.Title = "Library"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	return Model{list: l}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) SetNovels(novels []core.Novel) {
	m.novels = novels
	items := make([]list.Item, len(novels))
	for i, n := range novels {
		items[i] = item{novel: n}
	}
	m.list.SetItems(items)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if sel := m.list.SelectedItem(); sel != nil {
				if i, ok := sel.(item); ok {
					return m, func() tea.Msg {
						return struct{ NovelID string }{NovelID: i.novel.ID}
					}
				}
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) SetSize(width, height int) Model {
	m.width = width
	m.height = height
	m.list.SetSize(width, height)
	return m
}

func (m Model) View() string {
	if len(m.novels) == 0 {
		return lipgloss.NewStyle().Margin(1, 2).Render("Your library is empty. Press 's' to search and add novels.")
	}
	return m.list.View()
}

package chapterlist

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dharuncs/novel/internal/core"
)

type SelectChapterMsg struct {
	ChapterID string
}

type chapterItem struct {
	chapter core.Chapter
}

func (i chapterItem) FilterValue() string {
	return fmt.Sprintf("%g %s", i.chapter.Number, i.chapter.Title)
}

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(chapterItem)
	if !ok {
		return
	}

	style := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	if index == m.Index() {
		style = style.Foreground(lipgloss.Color("170")).Bold(true)
	}

	title := i.chapter.Title
	if title == "" {
		title = fmt.Sprintf("Chapter %g", i.chapter.Number)
	} else {
		title = fmt.Sprintf("Chapter %g: %s", i.chapter.Number, title)
	}

	fmt.Fprint(w, style.Render(title))
}

type Model struct {
	list   list.Model
	width  int
	height int
	novel  core.Novel
}

func New() Model {
	l := list.New([]list.Item{}, itemDelegate{}, 0, 0)
	l.Title = "Chapter List"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	return Model{list: l}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) SetChapters(novel core.Novel, chapters []core.Chapter) Model {
	m.novel = novel
	items := make([]list.Item, len(chapters))
	for i, ch := range chapters {
		items[i] = chapterItem{chapter: ch}
	}
	m.list.SetItems(items)
	m.list.Title = fmt.Sprintf("%s — Chapters (%d)", novel.Title, len(chapters))
	return m
}

func (m Model) SetSize(width, height int) Model {
	m.width = width
	m.height = height
	m.list.SetSize(width, height)
	return m
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if len(m.list.Items()) > 0 {
				if sel, ok := m.list.SelectedItem().(chapterItem); ok {
					chID := sel.chapter.ID
					return m, func() tea.Msg {
						return SelectChapterMsg{ChapterID: chID}
					}
				}
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return m.list.View()
}

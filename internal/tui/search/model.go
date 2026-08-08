package search

import (
	"context"
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dharuncs/novel/internal/source"
)

type AddNovelMsg struct {
	SourceID  string
	SourceURL string
}

type SearchResultsMsg struct {
	Results []source.SearchResult
	Err     error
}

type searchItem struct {
	result source.SearchResult
}

func (i searchItem) FilterValue() string { return i.result.Title }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 2 }
func (d itemDelegate) Spacing() int                              { return 1 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(searchItem)
	if !ok {
		return
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	authorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	if index == m.Index() {
		titleStyle = titleStyle.Foreground(lipgloss.Color("170")).Underline(true)
	}

	fmt.Fprintf(w, "%s\n%s",
		titleStyle.Render(i.result.Title),
		authorStyle.Render(i.result.Author),
	)
}

type Model struct {
	input     textinput.Model
	list      list.Model
	plugin    *source.Plugin
	searching bool
	statusMsg string
	err       error
	width     int
	height    int
}

func New() Model {
	ti := textinput.New()
	ti.Placeholder = "Search novel title..."
	ti.Focus()

	l := list.New([]list.Item{}, itemDelegate{}, 0, 0)
	l.Title = "Search Results"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)

	return Model{
		input: ti,
		list:  l,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) SetPlugin(p *source.Plugin) Model {
	m.plugin = p
	return m
}

func (m Model) Plugin() *source.Plugin {
	return m.plugin
}

func (m Model) SetSize(width, height int) Model {
	m.width = width
	m.height = height
	m.list.SetSize(width, height-6)
	return m
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case SearchResultsMsg:
		m.searching = false
		if msg.Err != nil {
			m.err = msg.Err
			m.statusMsg = fmt.Sprintf("Search error: %v", msg.Err)
			return m, nil
		}
		items := make([]list.Item, len(msg.Results))
		for i, res := range msg.Results {
			items[i] = searchItem{result: res}
		}
		m.list.SetItems(items)
		if len(items) > 0 {
			m.list.Select(0)
		}
		m.statusMsg = fmt.Sprintf("Found %d results (Press Down/j to select, Enter to add)", len(msg.Results))

	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.input.Focused() && m.input.Value() != "" {
				m.searching = true
				m.statusMsg = "Searching source..."
				query := m.input.Value()
				p := m.plugin
				searchCmd := func() tea.Msg {
					if p == nil {
						return SearchResultsMsg{Err: fmt.Errorf("no source plugin selected")}
					}
					res, err := p.Search(context.Background(), query, 1)
					return SearchResultsMsg{Results: res, Err: err}
				}
				m.input.Blur()
				return m, searchCmd
			} else if !m.input.Focused() && len(m.list.Items()) > 0 {
				if sel, ok := m.list.SelectedItem().(searchItem); ok {
					pID := ""
					if m.plugin != nil {
						pID = m.plugin.Metadata.ID
					}
					addCmd := func() tea.Msg {
						return AddNovelMsg{
							SourceID:  pID,
							SourceURL: sel.result.URL,
						}
					}
					return m, addCmd
				}
			}

		case "tab":
			if m.input.Focused() {
				m.input.Blur()
			} else {
				m.input.Focus()
			}
		}
	}

	if m.input.Focused() {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Padding(0, 1)
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Padding(0, 1)

	header := headerStyle.Render("Search Source")
	inputView := m.input.View()
	status := statusStyle.Render(m.statusMsg)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		" Search Query: "+inputView,
		status,
		m.list.View(),
	)
}

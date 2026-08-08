package search

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dharuncs/novel/internal/source"
)

func TestSearchModelSelection(t *testing.T) {
	model := New()
	plugin := &source.Plugin{
		Metadata: source.Metadata{
			ID:      "novelfire",
			Name:    "Novel Fire",
			BaseURL: "https://novelfire.net",
		},
	}
	model = model.SetPlugin(plugin)
	model = model.SetSize(80, 20)

	results := []source.SearchResult{
		{Title: "Lord of the Mysteries", Author: "Cuttlefish", URL: "https://example.com/lotm"},
		{Title: "Circle of Inevitability", Author: "Cuttlefish", URL: "https://example.com/coi"},
	}
	model, _ = model.Update(SearchResultsMsg{Results: results})

	model.input.Blur()

	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected command on item selection")
	}

	msg := cmd()
	addMsg, ok := msg.(AddNovelMsg)
	if !ok {
		t.Fatalf("expected AddNovelMsg on selection, got %T", msg)
	}

	if addMsg.SourceURL != "https://example.com/lotm" {
		t.Errorf("expected URL 'https://example.com/lotm', got %q", addMsg.SourceURL)
	}
}

func TestSearchModelSelection_KeyDownAndEnter(t *testing.T) {
	model := New()
	plugin := &source.Plugin{
		Metadata: source.Metadata{
			ID:      "novelfire",
			Name:    "Novel Fire",
			BaseURL: "https://novelfire.net",
		},
	}
	model = model.SetPlugin(plugin)
	model = model.SetSize(80, 20)

	// 1. Inject search results
	results := []source.SearchResult{
		{Title: "Lord of the Mysteries 1", Author: "Cuttlefish", URL: "https://novelfire.net/book/lotm-1"},
		{Title: "Lord of the Mysteries 2", Author: "Cuttlefish", URL: "https://novelfire.net/book/lotm-2"},
	}
	model, _ = model.Update(SearchResultsMsg{Results: results})

	// 2. Blur input
	model.input.Blur()

	// 3. Simulate down arrow keypress to move cursor to second item
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})

	// 4. Press enter to select second result
	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected command on enter keypress when input is blurred")
	}

	msg := cmd()
	addMsg, ok := msg.(AddNovelMsg)
	if !ok {
		t.Fatalf("expected AddNovelMsg on enter keypress, got %T", msg)
	}

	if addMsg.SourceID != "novelfire" {
		t.Errorf("expected SourceID 'novelfire', got %q", addMsg.SourceID)
	}
	if addMsg.SourceURL != "https://novelfire.net/book/lotm-2" {
		t.Errorf("expected selected URL 'https://novelfire.net/book/lotm-2', got %q", addMsg.SourceURL)
	}
}

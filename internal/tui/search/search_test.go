package search

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dharuncs/novel/internal/source"
)

func TestSearchModelSelection(t *testing.T) {
	model := New()
	model = model.SetSize(80, 20)

	// Simulate receiving SearchResultsMsg
	results := []source.SearchResult{
		{Title: "Lord of the Mysteries", Author: "Cuttlefish", URL: "https://example.com/lotm"},
		{Title: "Circle of Inevitability", Author: "Cuttlefish", URL: "https://example.com/coi"},
	}
	model, _ = model.Update(SearchResultsMsg{Results: results})

	if len(model.list.Items()) != 2 {
		t.Fatalf("expected 2 items in search list, got %d", len(model.list.Items()))
	}

	// Blur input so focus moves to list
	model.input.Blur()

	// Press enter on first search result
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

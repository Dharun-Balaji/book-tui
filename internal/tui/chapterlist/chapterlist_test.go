package chapterlist

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dharuncs/novel/internal/core"
)

func TestChapterListSelection(t *testing.T) {
	model := New()
	model = model.SetSize(80, 20)

	novel := core.Novel{ID: "n1", Title: "Test Novel"}
	chapters := []core.Chapter{
		{ID: "c1", Number: 1.0, Title: "Chapter 1"},
		{ID: "c2", Number: 2.0, Title: "Chapter 2"},
	}

	model = model.SetChapters(novel, chapters)
	if len(model.list.Items()) != 2 {
		t.Fatalf("expected 2 chapter items, got %d", len(model.list.Items()))
	}

	// Press enter on first chapter
	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected command on chapter selection")
	}

	msg := cmd()
	selectMsg, ok := msg.(SelectChapterMsg)
	if !ok {
		t.Fatalf("expected SelectChapterMsg, got %T", msg)
	}

	if selectMsg.ChapterID != "c1" {
		t.Errorf("expected selected ChapterID 'c1', got %q", selectMsg.ChapterID)
	}
}

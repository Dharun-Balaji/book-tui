package reader

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dharuncs/novel/internal/core"
)

func helperUnwrapBatch(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	res := cmd()
	if batch, ok := res.(tea.BatchMsg); ok {
		var msgs []tea.Msg
		for _, subCmd := range batch {
			if subCmd != nil {
				if subMsg := subCmd(); subMsg != nil {
					msgs = append(msgs, subMsg)
				}
			}
		}
		return msgs
	}
	if res != nil {
		return []tea.Msg{res}
	}
	return nil
}

func TestReaderAutoSaveOnScrollThreshold(t *testing.T) {
	ch := core.Chapter{
		ID:        "ch-1",
		NovelID:   "n-1",
		Number:    1.0,
		Title:     "Title 1",
		Content:   "P0 line A\n\nP1 line B\n\nP2 line C\n\nP3 line D\n\nP4 line E\n\nP5 line F\n\nP6 line G\n\nP7 line H\n\nP8 line I\n\nP9 line J",
		IsCached:  true,
		FetchedAt: nil,
	}

	model := New().SetChapter("n-1", ch, 0, 0, false, 80, 5)
	model = model.SetSize(80, 10)

	autoSaveFired := false
	var savedMsg SaveProgressMsg

	for i := 0; i < 30; i++ {
		var cmd tea.Cmd
		model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		msgs := helperUnwrapBatch(cmd)
		for _, m := range msgs {
			if saveMsg, ok := m.(SaveProgressMsg); ok {
				autoSaveFired = true
				savedMsg = saveMsg
				break
			}
		}
		if autoSaveFired {
			break
		}
	}

	if !autoSaveFired {
		t.Fatalf("expected auto-save SaveProgressMsg to fire after scrolling across threshold")
	}

	if savedMsg.NovelID != "n-1" || savedMsg.ChapterID != "ch-1" {
		t.Errorf("unexpected save message target: %#v", savedMsg)
	}
}

func TestReaderAutoSaveOnExit(t *testing.T) {
	ch := core.Chapter{
		ID:       "ch-1",
		NovelID:  "n-1",
		Number:   1.0,
		Title:    "Title 1",
		Content:  "P0 line A\n\nP1 line B",
		IsCached: true,
	}

	model := New().SetChapter("n-1", ch, 0, 0, false, 80, 5)
	model = model.SetSize(80, 10)

	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatalf("expected batch command on exit")
	}

	msgs := helperUnwrapBatch(cmd)

	hasSaveMsg := false
	hasSwitchViewMsg := false
	for _, msg := range msgs {
		if _, ok := msg.(SaveProgressMsg); ok {
			hasSaveMsg = true
		}
		if _, ok := msg.(SwitchToLibraryMsg); ok {
			hasSwitchViewMsg = true
		}
	}

	if !hasSaveMsg {
		t.Errorf("exit did not trigger SaveProgressMsg")
	}
	if !hasSwitchViewMsg {
		t.Errorf("exit did not trigger SwitchToLibraryMsg")
	}
	_ = updatedModel
}

func TestReaderResizeParagraphAnchoredPosition(t *testing.T) {
	ch := core.Chapter{
		ID:       "ch-1",
		NovelID:  "n-1",
		Number:   1.0,
		Title:    "Title 1",
		Content:  "Paragraph 0 text long long text\n\nParagraph 1 text long long text\n\nParagraph 2 text",
		IsCached: true,
	}

	model := New().SetChapter("n-1", ch, 1, 0, false, 80, 5)
	model = model.SetSize(80, 10)

	if model.paragraphIdx != 1 {
		t.Errorf("expected paragraphIdx 1, got %d", model.paragraphIdx)
	}

	model = model.SetSize(40, 10)

	if model.paragraphIdx != 1 {
		t.Errorf("expected paragraphIdx 1 after resize, got %d", model.paragraphIdx)
	}
}

func TestBookmarkStub(t *testing.T) {
	ch := core.Chapter{ID: "ch-1", NovelID: "n-1", Title: "T1", Content: "P1"}
	model := New().SetChapter("n-1", ch, 0, 0, false, 80, 5)

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	if model.statusMsg != "Bookmarks not yet implemented in storage" {
		t.Errorf("expected bookmark stub message, got %q", model.statusMsg)
	}
}

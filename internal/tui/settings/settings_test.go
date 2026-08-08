package settings

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dharuncs/novel/internal/storage"
)

func TestSettingsToggleTheme(t *testing.T) {
	model := New()
	model = model.SetSettings(storage.UserSettings{
		Theme:         "dark",
		LineWidth:     80,
		AutoSaveEvery: 5,
	})

	// Press enter on cursor 0 (Theme)
	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected command on theme toggle")
	}

	msg := cmd()
	saveMsg, ok := msg.(SaveSettingsMsg)
	if !ok {
		t.Fatalf("expected SaveSettingsMsg, got %T", msg)
	}

	if saveMsg.Settings.Theme != "light" {
		t.Errorf("expected toggled theme to be 'light', got %q", saveMsg.Settings.Theme)
	}

	if updatedModel.settings.Theme != "light" {
		t.Errorf("expected model theme to update to 'light'")
	}
}

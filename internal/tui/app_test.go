package tui

import (
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dharuncs/novel/internal/core"
	"github.com/dharuncs/novel/internal/source"
	"github.com/dharuncs/novel/internal/storage"
	"github.com/dharuncs/novel/internal/tui/reader"
	"github.com/dharuncs/novel/internal/tui/search"
	"github.com/dharuncs/novel/internal/tui/settings"
)

func unwrapAllMsgs(res tea.Msg) []tea.Msg {
	if res == nil {
		return nil
	}
	if batch, ok := res.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, subCmd := range batch {
			if subCmd != nil {
				out = append(out, unwrapAllMsgs(subCmd())...)
			}
		}
		return out
	}
	return []tea.Msg{res}
}

func setupTestApp(t *testing.T, dbName string) (*storage.DB, AppModel, storage.Novel, storage.Chapter) {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), dbName))
	if err != nil {
		t.Fatalf("storage open error: %v", err)
	}

	now := time.Now().UTC()
	if err := db.UpsertSource(storage.Source{
		ID:        "src1",
		Name:      "Source 1",
		Version:   "1.0.0",
		BaseURL:   "https://example.com",
		Language:  "en",
		RateLimit: 60,
	}); err != nil {
		t.Fatalf("upsert source: %v", err)
	}

	novel := storage.Novel{
		ID:            "novel-p-1",
		SourceID:      "src1",
		SourceURL:     "https://example.com/novel-p1",
		Title:         "Persistence Test Novel",
		Author:        "Author P",
		Status:        "ongoing",
		TotalChapters: 5,
		InLibrary:     true,
		AddedAt:       now,
		UpdatedAt:     now,
	}
	if err := db.CreateNovel(novel); err != nil {
		t.Fatalf("create novel: %v", err)
	}

	content := "P0 line A\n\nP1 line B\n\nP2 line C\n\nP3 line D\n\nP4 line E\n\nP5 line F\n\nP6 line G\n\nP7 line H\n\nP8 line I\n\nP9 line J"
	chapter := storage.Chapter{
		ID:        "ch-p-1",
		NovelID:   novel.ID,
		SourceURL: "https://example.com/ch-p1",
		Number:    1.0,
		Title:     "Chapter 1",
		Content:   content,
		IsCached:  true,
		FetchedAt: &now,
	}
	if err := db.CreateChapter(chapter); err != nil {
		t.Fatalf("create chapter: %v", err)
	}

	lm := core.NewLibraryManager(db)
	pm := core.NewProgressManager(db)
	sm := core.NewStatsManager(db)
	reg := source.NewRegistry()

	app := NewAppModel(lm, pm, sm, reg, db)
	return db, app, novel, chapter
}

func TestAppModelResumePositionWiring(t *testing.T) {
	db, app, novel, chapter := setupTestApp(t, "tui_test_resume.db")
	defer db.Close()

	now := time.Now().UTC()
	progress := storage.ReadingProgress{
		NovelID:      novel.ID,
		ChapterID:    chapter.ID,
		ChapterNum:   1.0,
		ParagraphIdx: 7,
		ScrollOffset: 14,
		ProgressPct:  0.5,
		LastReadAt:   now,
	}
	if err := db.SaveProgress(progress); err != nil {
		t.Fatal(err)
	}

	cmd := app.openNovelCmd(novel.ID)
	msg := cmd()

	loadedMsg, ok := msg.(ChapterLoadedMsg)
	if !ok {
		t.Fatalf("expected ChapterLoadedMsg, got %T", msg)
	}
	if loadedMsg.ParagraphIdx != 7 || loadedMsg.ScrollOffset != 14 {
		t.Errorf("expected position 7/14, got %d/%d", loadedMsg.ParagraphIdx, loadedMsg.ScrollOffset)
	}

	updatedApp, _ := app.Update(loadedMsg)
	appModel := updatedApp.(AppModel)
	if appModel.state != ViewReader {
		t.Errorf("expected ViewReader state after ChapterLoadedMsg")
	}
}

func TestAppModelPersistence_ThresholdScroll(t *testing.T) {
	db, app, novel, chapter := setupTestApp(t, "tui_test_threshold.db")
	defer db.Close()

	longContent := ""
	for i := 0; i < 30; i++ {
		longContent += "Paragraph line text for scrolling test\n\n"
	}
	chapter.Content = longContent
	_ = db.CreateChapter(chapter)

	updatedApp, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	app = updatedApp.(AppModel)

	loadedMsg := ChapterLoadedMsg{
		Chapter:         chapter,
		ParagraphIdx:    0,
		ScrollOffset:    0,
		IsNovelComplete: false,
	}
	app.activeNovelID = novel.ID
	updatedApp, _ = app.Update(loadedMsg)
	appModel := updatedApp.(AppModel)

	// Scroll down line by line to trigger paragraph threshold auto-save
	for i := 0; i < 30; i++ {
		var cmd tea.Cmd
		updatedApp, cmd = appModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		appModel = updatedApp.(AppModel)
		if cmd != nil {
			msgs := unwrapAllMsgs(cmd())
			for _, subMsg := range msgs {
				if saveMsg, ok := subMsg.(reader.SaveProgressMsg); ok {
					_, saveCmd := appModel.Update(saveMsg)
					if saveCmd != nil {
						saveCmd() // Executes db.SaveProgress synchronously inside test
					}
				}
			}
		}
	}

	// Query DB DIRECTLY to verify persistence
	dbProg, err := db.GetProgress(novel.ID)
	if err != nil {
		t.Fatalf("failed to query DB directly: %v", err)
	}
	if dbProg == nil {
		t.Fatalf("expected progress record in DB, got nil")
	}
	if dbProg.ParagraphIdx == 0 {
		t.Errorf("expected ParagraphIdx > 0 in DB after scrolling threshold, got %d", dbProg.ParagraphIdx)
	}
}

func TestAppModelPersistence_BackNavigation(t *testing.T) {
	db, app, novel, chapter := setupTestApp(t, "tui_test_back.db")
	defer db.Close()

	updatedApp, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	app = updatedApp.(AppModel)

	loadedMsg := ChapterLoadedMsg{
		Chapter:         chapter,
		ParagraphIdx:    3,
		ScrollOffset:    1,
		IsNovelComplete: false,
	}
	app.activeNovelID = novel.ID
	updatedApp, _ = app.Update(loadedMsg)
	appModel := updatedApp.(AppModel)

	// Press 'q' to go back to library
	updatedApp, cmd := appModel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	appModel = updatedApp.(AppModel)

	if cmd != nil {
		msgs := unwrapAllMsgs(cmd())
		for _, msg := range msgs {
			if saveMsg, ok := msg.(reader.SaveProgressMsg); ok {
				_, saveCmd := appModel.Update(saveMsg)
				if saveCmd != nil {
					saveCmd()
				}
			}
		}
	}

	// Query DB DIRECTLY to verify back-navigation save
	dbProg, err := db.GetProgress(novel.ID)
	if err != nil {
		t.Fatalf("failed to query DB directly: %v", err)
	}
	if dbProg == nil {
		t.Fatalf("expected progress record in DB, got nil")
	}
	if dbProg.ParagraphIdx != 3 {
		t.Errorf("expected ParagraphIdx 3 in DB after back navigation, got %d", dbProg.ParagraphIdx)
	}
}

func TestAppModelPersistence_CtrlCQuit(t *testing.T) {
	db, app, novel, chapter := setupTestApp(t, "tui_test_quit.db")
	defer db.Close()

	updatedApp, _ := app.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
	app = updatedApp.(AppModel)

	loadedMsg := ChapterLoadedMsg{
		Chapter:         chapter,
		ParagraphIdx:    4,
		ScrollOffset:    2,
		IsNovelComplete: false,
	}
	app.activeNovelID = novel.ID
	updatedApp, _ = app.Update(loadedMsg)
	appModel := updatedApp.(AppModel)

	// Press 'ctrl+c' while in ViewReader state
	_, _ = appModel.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	// Query DB DIRECTLY to verify ctrl+c flushed progress to disk
	dbProg, err := db.GetProgress(novel.ID)
	if err != nil {
		t.Fatalf("failed to query DB directly: %v", err)
	}
	if dbProg == nil {
		t.Fatalf("expected progress record in DB, got nil")
	}
	if dbProg.ParagraphIdx != 4 {
		t.Errorf("expected ParagraphIdx 4 in DB after ctrl+c quit, got %d", dbProg.ParagraphIdx)
	}
}

func TestAppModelChapterNavigation(t *testing.T) {
	db, app, novel, ch1 := setupTestApp(t, "tui_test_nav.db")
	defer db.Close()

	now := time.Now().UTC()
	ch2 := storage.Chapter{
		ID:        "ch-p-2",
		NovelID:   novel.ID,
		SourceURL: "https://example.com/ch-p2",
		Number:    2.0,
		Title:     "Chapter 2",
		Content:   "Chapter 2 content text",
		IsCached:  true,
		FetchedAt: &now,
	}
	_ = db.CreateChapter(ch2)

	app.activeNovelID = novel.ID
	loadedMsg := ChapterLoadedMsg{
		Chapter:         ch1,
		ParagraphIdx:    0,
		ScrollOffset:    0,
		IsNovelComplete: false,
	}
	updatedApp, _ := app.Update(loadedMsg)
	appModel := updatedApp.(AppModel)

	// Send NextChapterMsg
	updatedApp, cmd := appModel.Update(reader.NextChapterMsg{NovelID: novel.ID, CurrentChapterNum: 1.0})
	appModel = updatedApp.(AppModel)
	if cmd == nil {
		t.Fatalf("expected command for NextChapterMsg")
	}

	subMsg := cmd()
	loadedNext, ok := subMsg.(ChapterLoadedMsg)
	if !ok {
		t.Fatalf("expected ChapterLoadedMsg for Chapter 2, got %T", subMsg)
	}

	if loadedNext.Chapter.ID != ch2.ID {
		t.Errorf("expected loaded chapter ID %q, got %q", ch2.ID, loadedNext.Chapter.ID)
	}
}

func TestAppModelPersistence_SettingsRestart(t *testing.T) {
	db, app, _, _ := setupTestApp(t, "tui_test_settings_restart.db")
	defer db.Close()

	newSettings := storage.UserSettings{
		Theme:         "light",
		LineWidth:     100,
		AutoSaveEvery: 10,
	}

	_, cmd := app.Update(settings.SaveSettingsMsg{Settings: newSettings})
	if cmd != nil {
		cmd()
	}

	dbSet, err := db.GetSettings()
	if err != nil {
		t.Fatalf("failed to query DB directly for settings: %v", err)
	}
	if dbSet.Theme != "light" || dbSet.LineWidth != 100 || dbSet.AutoSaveEvery != 10 {
		t.Errorf("unexpected settings persisted in DB: %#v", dbSet)
	}

	lm := core.NewLibraryManager(db)
	pm := core.NewProgressManager(db)
	sm := core.NewStatsManager(db)
	reg := source.NewRegistry()

	restartedApp := NewAppModel(lm, pm, sm, reg, db)

	if restartedApp.userSettings.Theme != "light" {
		t.Errorf("expected restarted app to load theme 'light', got %q", restartedApp.userSettings.Theme)
	}
	if restartedApp.userSettings.LineWidth != 100 {
		t.Errorf("expected restarted app to load LineWidth 100, got %d", restartedApp.userSettings.LineWidth)
	}
}

func TestSearchSelectionToLibraryAddPipeline(t *testing.T) {
	db, app, _, _ := setupTestApp(t, "tui_test_search_pipeline.db")
	defer db.Close()

	// 1. Create a mock source plugin and register it
	p := &source.Plugin{
		Metadata: source.Metadata{
			ID:      "src1",
			Name:    "Source 1",
			BaseURL: "https://example.com",
		},
	}
	_ = app.registry.Add(p)

	// 2. Press 's' to enter ViewSearch
	updatedApp, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	appModel := updatedApp.(AppModel)
	if appModel.state != ViewSearch {
		t.Fatalf("expected ViewSearch state after pressing 's', got %v", appModel.state)
	}

	// 3. Inject SearchResultsMsg
	searchResults := []source.SearchResult{
		{Title: "Searched Novel 1", Author: "Author 1", URL: "https://example.com/novel-p1"},
		{Title: "Searched Novel 2", Author: "Author 2", URL: "https://example.com/novel-p2"},
	}
	updatedApp, _ = appModel.Update(search.SearchResultsMsg{Results: searchResults})
	appModel = updatedApp.(AppModel)

	// 4. Blur search text input to move focus to list
	appModel.searchModel, _ = appModel.searchModel.Update(search.SearchResultsMsg{})

	// 5. Select item and dispatch AddNovelMsg
	addMsg := search.AddNovelMsg{
		SourceID:  "src1",
		SourceURL: "https://example.com/novel-p1",
	}

	updatedApp, cmd := appModel.Update(addMsg)
	appModel = updatedApp.(AppModel)

	// 6. Assert app transitions to ViewLibrary immediately
	if appModel.state != ViewLibrary {
		t.Errorf("expected ViewLibrary state after AddNovelMsg, got %v", appModel.state)
	}

	if cmd == nil {
		t.Fatalf("expected addNovelCmd execution")
	}

	// 7. Verify novel is in library DB directly
	inLib, _, err := appModel.libraryManager.IsNovelInLibrary("src1", "https://example.com/novel-p1")
	if err != nil || !inLib {
		t.Errorf("expected novel to be in library, got inLib=%v, err=%v", inLib, err)
	}
}

func TestAppModelWindowSizeHandling(t *testing.T) {
	db, app, _, _ := setupTestApp(t, "tui_test_size.db")
	defer db.Close()

	updatedApp, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	appModel := updatedApp.(AppModel)

	if appModel.width != 100 || appModel.height != 40 {
		t.Errorf("expected dimensions 100x40, got %dx%d", appModel.width, appModel.height)
	}
}

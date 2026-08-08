package core

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dharuncs/novel/internal/scraper"
	"github.com/dharuncs/novel/internal/source"
	"github.com/dharuncs/novel/internal/storage"
)

func testCoreDB(t *testing.T) *storage.DB {
	t.Helper()
	database, err := storage.Open(filepath.Join(t.TempDir(), "core_test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

func mockPlugin(t *testing.T) *source.Plugin {
	t.Helper()
	script := `const source = {
		id: "mock", name: "Mock Source", version: "1.0", baseURL: "https://example.com", language: "en", rateLimit: 60, needsJS: false,
		novelInfo: function(url) {
			return { url: url, title: "Mock Novel", author: "Mock Author", coverURL: "https://example.com/cover.png", description: "Desc", status: "ongoing", tags: ["action"], totalChapters: 2 };
		},
		chapterList: function(url) {
			return [
				{ url: "https://example.com/ch1", title: "Chapter 1", number: 1.0 },
				{ url: "https://example.com/ch2", title: "Chapter 2", number: 2.0 }
			];
		},
		search: function(q, p) { return []; },
		chapterContent: function(url) { return "<p>Line 1 &amp; <b>bold</b></p><br/><p>Line 2</p>"; }
	};`
	plugin, err := source.LoadScript(script, scraper.NewClient())
	if err != nil {
		t.Fatalf("load mock plugin: %v", err)
	}
	return plugin
}

func TestLibraryManager(t *testing.T) {
	db := testCoreDB(t)
	lm := NewLibraryManager(db)
	plugin := mockPlugin(t)
	ctx := context.Background()

	url := "https://example.com/novel"

	inLib, _, err := lm.IsNovelInLibrary("mock", url)
	if err != nil || inLib {
		t.Fatalf("expected not in library, got inLib=%v, err=%v", inLib, err)
	}

	var progressLogs []string
	novel, err := lm.AddNovel(ctx, plugin, url, func(status string) {
		progressLogs = append(progressLogs, status)
	})
	t.Logf("Captured progress logs: %#v", progressLogs)
	if err != nil {
		t.Fatalf("AddNovel failed: %v", err)
	}
	if len(progressLogs) < 3 {
		t.Fatalf("expected at least 3 progress logs, got %d: %v", len(progressLogs), progressLogs)
	}
	hasSavingStep := false
	for _, logMsg := range progressLogs {
		if strings.Contains(logMsg, "Saving novel") {
			hasSavingStep = true
			break
		}
	}
	if !hasSavingStep {
		t.Fatalf("progress logs missing expected saving step: %v", progressLogs)
	}
	if novel.Title != "Mock Novel" || novel.Author != "Mock Author" || novel.TotalChapters != 2 {
		t.Fatalf("unexpected novel fields: %#v", novel)
	}

	lib, err := lm.ListLibrary()
	if err != nil || len(lib) != 1 {
		t.Fatalf("expected 1 novel in library, got %d, err=%v", len(lib), err)
	}

	chapters, err := lm.ListChapters(novel.ID)
	if err != nil || len(chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d, err=%v", len(chapters), err)
	}

	// Verify content cleaning on fetch
	ch1, err := lm.GetChapterContent(ctx, plugin, chapters[0].ID)
	if err != nil || !ch1.IsCached || ch1.Content != "Line 1 & **bold**\nLine 2" {
		t.Fatalf("GetChapterContent failed: %#v, err=%v", ch1, err)
	}

	if err := lm.RemoveNovel(novel.ID); err != nil {
		t.Fatalf("RemoveNovel failed: %v", err)
	}
}

func TestProgressManager(t *testing.T) {
	db := testCoreDB(t)
	lm := NewLibraryManager(db)
	pm := NewProgressManager(db)
	plugin := mockPlugin(t)
	ctx := context.Background()

	novel, err := lm.AddNovel(ctx, plugin, "https://example.com/novel", nil)
	if err != nil {
		t.Fatal(err)
	}
	chapters, err := lm.ListChapters(novel.ID)
	if err != nil || len(chapters) < 2 {
		t.Fatal("expected chapters")
	}

	// 1. Initial state (no progress) -> Chapter 1, IsNovelComplete=false
	state, err := pm.GetResumeState(novel.ID)
	if err != nil || state.Progress != nil || state.Chapter.ID != chapters[0].ID || state.IsNovelComplete {
		t.Fatalf("initial resume state: %#v, err=%v", state, err)
	}

	// 2. Partial reading on Chapter 1 -> Chapter 1, IsNovelComplete=false
	if err := pm.SaveReadingState(novel.ID, chapters[0].ID, 2, 0, 10, 120); err != nil {
		t.Fatalf("SaveReadingState error: %v", err)
	}
	state, err = pm.GetResumeState(novel.ID)
	if err != nil || state.Progress == nil || state.Chapter.ID != chapters[0].ID || state.IsNovelComplete {
		t.Fatalf("partial resume state failed: %#v, err=%v", state, err)
	}

	// 3. 100% complete on Chapter 1 -> suggests Chapter 2, IsNovelComplete=false
	if err := pm.SaveReadingState(novel.ID, chapters[0].ID, 10, 0, 10, 300); err != nil {
		t.Fatalf("SaveReadingState 100%% error: %v", err)
	}
	state, err = pm.GetResumeState(novel.ID)
	if err != nil || state.Chapter.ID != chapters[1].ID || state.IsNovelComplete {
		t.Fatalf("completed chapter 1 state failed: %#v, err=%v", state, err)
	}

	// 4. 100% complete on Chapter 2 (the LAST chapter) -> IsNovelComplete=true!
	if err := pm.SaveReadingState(novel.ID, chapters[1].ID, 10, 0, 10, 300); err != nil {
		t.Fatalf("SaveReadingState ch2 100%% error: %v", err)
	}
	state, err = pm.GetResumeState(novel.ID)
	if err != nil || state.Chapter.ID != chapters[1].ID || !state.IsNovelComplete {
		t.Fatalf("completed LAST chapter state failed: %#v, err=%v", state, err)
	}
}

func TestStatsManager(t *testing.T) {
	db := testCoreDB(t)
	lm := NewLibraryManager(db)
	pm := NewProgressManager(db)
	sm := NewStatsManager(db)
	plugin := mockPlugin(t)
	ctx := context.Background()

	novel, err := lm.AddNovel(ctx, plugin, "https://example.com/novel", nil)
	if err != nil {
		t.Fatal(err)
	}
	chapters, _ := lm.ListChapters(novel.ID)

	if err := pm.SaveReadingState(novel.ID, chapters[0].ID, 10, 0, 10, 180); err != nil {
		t.Fatal(err)
	}

	session, err := sm.StartReadingSession(novel.ID, chapters[0].ID)
	if err != nil {
		t.Fatalf("StartReadingSession error: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := sm.EndReadingSession(session.ID); err != nil {
		t.Fatalf("EndReadingSession error: %v", err)
	}

	overall, err := sm.GetOverallStats()
	if err != nil {
		t.Fatalf("GetOverallStats error: %v", err)
	}
	if overall.TotalLibraryNovels != 1 || overall.TotalChaptersRead != 1 || overall.TotalSessions != 1 {
		t.Fatalf("unexpected overall stats: %#v", overall)
	}
}

func TestCleanContent(t *testing.T) {
	raw := "  <p>Hello &amp; <b>World</b></p><br/><p>Second line</p>  "
	cleaned := CleanContent(raw)
	expected := "Hello & **World**\nSecond line"
	if cleaned != expected {
		t.Errorf("CleanContent expected %q, got %q", expected, cleaned)
	}
}

func TestGetChapterContent_DoNotCacheEmptyContent(t *testing.T) {
	db := testCoreDB(t)
	lm := NewLibraryManager(db)
	now := time.Now().UTC()

	_ = db.UpsertSource(storage.Source{
		ID:        "src1",
		Name:      "Src",
		Version:   "1.0",
		BaseURL:   "https://example.com",
		Language:  "en",
		RateLimit: 60,
	})

	novel := Novel{
		ID:        "n1",
		SourceID:  "src1",
		SourceURL: "https://example.com/novel",
		Title:     "Test Novel",
		Status:    "ongoing",
		AddedAt:   now,
		UpdatedAt: now,
	}
	if err := db.CreateNovel(novel); err != nil {
		t.Fatal(err)
	}

	chStub := Chapter{
		ID:        "c1",
		NovelID:   "n1",
		SourceURL: "https://example.com/ch1",
		Number:    1.0,
		Title:     "Chapter 1",
	}
	if err := db.CreateChapter(chStub); err != nil {
		t.Fatal(err)
	}

	script := `const source = {
		id: "src1", name: "Src", version: "1.0", baseURL: "https://example.com", language: "en", rateLimit: 60, needsJS: false,
		novelInfo: function(url) { return {}; },
		chapterList: function(url) { return []; },
		search: function(q, p) { return []; },
		chapterContent: function(url) { return ""; }
	};`
	emptyPlugin, err := source.LoadScript(script, scraper.NewClient())
	if err != nil {
		t.Fatal(err)
	}

	_, err = lm.GetChapterContent(context.Background(), emptyPlugin, "c1")
	if err == nil {
		t.Fatalf("expected error when fetching empty chapter content")
	}

	dbCh, err := db.GetChapter("c1")
	if err != nil {
		t.Fatalf("failed to query DB for chapter: %v", err)
	}
	if dbCh.IsCached {
		t.Errorf("expected IsCached to remain false when content is empty, got true")
	}
}

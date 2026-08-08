package core

import (
	"context"
	"path/filepath"
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
		chapterContent: function(url) { return "Line 1\n\nLine 2"; }
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

	// 1. Check initially not in library
	inLib, _, err := lm.IsNovelInLibrary("mock", url)
	if err != nil || inLib {
		t.Fatalf("expected not in library, got inLib=%v, err=%v", inLib, err)
	}

	// 2. Add novel to library
	novel, err := lm.AddNovel(ctx, plugin, url)
	if err != nil {
		t.Fatalf("AddNovel failed: %v", err)
	}
	if novel.Title != "Mock Novel" || novel.Author != "Mock Author" || novel.TotalChapters != 2 {
		t.Fatalf("unexpected novel fields: %#v", novel)
	}

	// 3. List library
	lib, err := lm.ListLibrary()
	if err != nil || len(lib) != 1 {
		t.Fatalf("expected 1 novel in library, got %d, err=%v", len(lib), err)
	}

	// 4. List chapters
	chapters, err := lm.ListChapters(novel.ID)
	if err != nil || len(chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d, err=%v", len(chapters), err)
	}

	// 5. Get chapter content (fetches from plugin & caches)
	ch1, err := lm.GetChapterContent(ctx, plugin, chapters[0].ID)
	if err != nil || !ch1.IsCached || ch1.Content != "Line 1\n\nLine 2" {
		t.Fatalf("GetChapterContent failed: %#v, err=%v", ch1, err)
	}

	// 6. Second call returns cached content directly
	ch1Cached, err := lm.GetChapterContent(ctx, plugin, chapters[0].ID)
	if err != nil || ch1Cached.Content != "Line 1\n\nLine 2" {
		t.Fatalf("cached GetChapterContent failed: %#v, err=%v", ch1Cached, err)
	}

	// 7. Remove novel from library
	if err := lm.RemoveNovel(novel.ID); err != nil {
		t.Fatalf("RemoveNovel failed: %v", err)
	}
	libAfter, err := lm.ListLibrary()
	if err != nil || len(libAfter) != 0 {
		t.Fatalf("expected 0 novels after remove, got %d", len(libAfter))
	}
}

func TestProgressManager(t *testing.T) {
	db := testCoreDB(t)
	lm := NewLibraryManager(db)
	pm := NewProgressManager(db)
	plugin := mockPlugin(t)
	ctx := context.Background()

	novel, err := lm.AddNovel(ctx, plugin, "https://example.com/novel")
	if err != nil {
		t.Fatal(err)
	}
	chapters, err := lm.ListChapters(novel.ID)
	if err != nil || len(chapters) < 2 {
		t.Fatal("expected chapters")
	}

	// 1. Initial state (no progress recorded yet) -> suggests chapter 1
	prog, targetCh, err := pm.GetResumeState(novel.ID)
	if err != nil || prog != nil || targetCh.ID != chapters[0].ID {
		t.Fatalf("initial resume state: prog=%v, ch=%v, err=%v", prog, targetCh, err)
	}

	// 2. Save partial reading state on chapter 1
	if err := pm.SaveReadingState(novel.ID, chapters[0].ID, 2, 0, 10, 120); err != nil {
		t.Fatalf("SaveReadingState error: %v", err)
	}
	prog, targetCh, err = pm.GetResumeState(novel.ID)
	if err != nil || prog == nil || targetCh.ID != chapters[0].ID || prog.ParagraphIdx != 2 {
		t.Fatalf("partial resume state failed: prog=%#v, ch=%#v, err=%v", prog, targetCh, err)
	}

	// 3. Save 100% complete reading state on chapter 1 -> suggests next chapter (chapter 2)
	if err := pm.SaveReadingState(novel.ID, chapters[0].ID, 10, 0, 10, 300); err != nil {
		t.Fatalf("SaveReadingState 100%% error: %v", err)
	}
	prog, targetCh, err = pm.GetResumeState(novel.ID)
	if err != nil || prog == nil || targetCh.ID != chapters[1].ID {
		t.Fatalf("completed chapter resume state failed: targetCh=%#v, err=%v", targetCh, err)
	}
}

func TestStatsManager(t *testing.T) {
	db := testCoreDB(t)
	lm := NewLibraryManager(db)
	pm := NewProgressManager(db)
	sm := NewStatsManager(db)
	plugin := mockPlugin(t)
	ctx := context.Background()

	novel, err := lm.AddNovel(ctx, plugin, "https://example.com/novel")
	if err != nil {
		t.Fatal(err)
	}
	chapters, _ := lm.ListChapters(novel.ID)

	// Save reading state (chapter 1 complete)
	if err := pm.SaveReadingState(novel.ID, chapters[0].ID, 10, 0, 10, 180); err != nil {
		t.Fatal(err)
	}

	// Start and end session
	session, err := sm.StartReadingSession(novel.ID, chapters[0].ID)
	if err != nil {
		t.Fatalf("StartReadingSession error: %v", err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := sm.EndReadingSession(session.ID); err != nil {
		t.Fatalf("EndReadingSession error: %v", err)
	}

	// Verify overall stats
	overall, err := sm.GetOverallStats()
	if err != nil {
		t.Fatalf("GetOverallStats error: %v", err)
	}
	if overall.TotalLibraryNovels != 1 || overall.TotalChaptersRead != 1 || overall.TotalSessions != 1 || overall.TotalReadingTime != 180*time.Second {
		t.Fatalf("unexpected overall stats: %#v", overall)
	}

	// Verify per-novel stats
	novelStats, err := sm.GetNovelStats(novel.ID)
	if err != nil {
		t.Fatalf("GetNovelStats error: %v", err)
	}
	if novelStats.NovelID != novel.ID || novelStats.ChaptersRead != 1 || novelStats.SessionCount != 1 {
		t.Fatalf("unexpected novel stats: %#v", novelStats)
	}
}

func TestProgressHelpers(t *testing.T) {
	if pct := CalculateProgressPct(0, 10); pct != 0.1 {
		t.Errorf("expected 0.1, got %f", pct)
	}
	if pct := CalculateProgressPct(9, 10); pct != 1.0 {
		t.Errorf("expected 1.0, got %f", pct)
	}
	if fmtStr := FormatProgressString(1.5, 0.456); fmtStr != "Ch. 1.5 (46%)" {
		t.Errorf("expected 'Ch. 1.5 (46%%)', got %q", fmtStr)
	}
}

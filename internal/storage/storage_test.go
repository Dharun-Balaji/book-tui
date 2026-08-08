package storage

import (
	"path/filepath"
	"testing"
	"time"
)

func testDB(t *testing.T) *DB {
	t.Helper()
	database, err := Open(filepath.Join(t.TempDir(), "novel.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}
func seed(t *testing.T, db *DB) (Novel, Chapter) {
	t.Helper()
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	if err := db.UpsertSource(Source{ID: "rr", Name: "Royal Road", Version: "1", BaseURL: "https://royalroad.com", Language: "en", RateLimit: 60}); err != nil {
		t.Fatal(err)
	}
	novel := Novel{ID: "novel-1", SourceID: "rr", SourceURL: "https://royalroad.com/fictions/1", Title: "First", Status: "ongoing", Tags: []string{"action", "fantasy"}, InLibrary: true, AddedAt: now, UpdatedAt: now}
	if err := db.CreateNovel(novel); err != nil {
		t.Fatal(err)
	}
	chapter := Chapter{ID: "chapter-1", NovelID: novel.ID, SourceURL: "https://royalroad.com/chapter/1", Number: 1, Title: "One"}
	if err := db.CreateChapter(chapter); err != nil {
		t.Fatal(err)
	}
	return novel, chapter
}
func TestNovelCRUD(t *testing.T) {
	db := testDB(t)
	novel, _ := seed(t, db)
	got, err := db.GetNovel(novel.ID)
	if err != nil || len(got.Tags) != 2 || got.Title != "First" {
		t.Fatalf("get novel: %#v, %v", got, err)
	}
	novel.Title = "Updated"
	novel.Tags = []string{"fantasy"}
	novel.UpdatedAt = novel.UpdatedAt.Add(time.Hour)
	if err = db.UpdateNovel(novel); err != nil {
		t.Fatal(err)
	}
	got, err = db.GetNovel(novel.ID)
	if err != nil || got.Title != "Updated" || len(got.Tags) != 1 {
		t.Fatalf("update novel: %#v, %v", got, err)
	}
	if err = db.DeleteNovel(novel.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.GetNovel(novel.ID); err == nil {
		t.Fatal("expected deleted novel lookup to fail")
	}
}

func TestSourceUpsertAndMigrations(t *testing.T) {
	db := testDB(t)
	var version int
	if err := db.SQL().QueryRow("SELECT version FROM schema_version").Scan(&version); err != nil || version != 2 {
		t.Fatalf("schema version: %d, %v", version, err)
	}
	if err := db.UpsertSource(Source{ID: "source", Name: "Original", Version: "1", BaseURL: "https://example.com", Language: "en", RateLimit: 30}); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertSource(Source{ID: "source", Name: "Updated", Version: "2", BaseURL: "https://example.com", Language: "en", NeedsJS: true, RateLimit: 60}); err != nil {
		t.Fatal(err)
	}
	var name, versionText string
	var needsJS bool
	if err := db.SQL().QueryRow("SELECT name, version, needs_js FROM sources WHERE id='source'").Scan(&name, &versionText, &needsJS); err != nil || name != "Updated" || versionText != "2" || !needsJS {
		t.Fatalf("source upsert: %q, %q, %t, %v", name, versionText, needsJS, err)
	}
}
func TestChapterCRUD(t *testing.T) {
	db := testDB(t)
	_, chapter := seed(t, db)
	fetched := time.Now().UTC()
	chapter.Title = "Updated"
	chapter.Content = "content"
	chapter.WordCount = 1
	chapter.FetchedAt = &fetched
	chapter.IsCached = true
	if err := db.UpdateChapter(chapter); err != nil {
		t.Fatal(err)
	}
	got, err := db.GetChapter(chapter.ID)
	if err != nil || got.Content != "content" || got.FetchedAt == nil {
		t.Fatalf("get chapter: %#v, %v", got, err)
	}
	if err = db.DeleteChapter(chapter.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = db.GetChapter(chapter.ID); err == nil {
		t.Fatal("expected deleted chapter lookup to fail")
	}
}
func TestProgressUpsert(t *testing.T) {
	db := testDB(t)
	novel, chapter := seed(t, db)
	progress := ReadingProgress{NovelID: novel.ID, ChapterID: chapter.ID, ChapterNum: 1, ParagraphIdx: 3, LastReadAt: time.Now().UTC()}
	if err := db.SaveProgress(progress); err != nil {
		t.Fatal(err)
	}
	progress.ParagraphIdx = 9
	progress.ChaptersRead = 1
	if err := db.SaveProgress(progress); err != nil {
		t.Fatal(err)
	}
	got, err := db.GetProgress(novel.ID)
	if err != nil || got.ParagraphIdx != 9 || got.ChaptersRead != 1 {
		t.Fatalf("upsert progress: %#v, %v", got, err)
	}
	var count int
	if err = db.SQL().QueryRow("SELECT count(*) FROM reading_progress").Scan(&count); err != nil || count != 1 {
		t.Fatalf("progress rows: %d, %v", count, err)
	}
}
func TestHistoryCRUD(t *testing.T) {
	db := testDB(t)
	novel, chapter := seed(t, db)
	entry := HistoryEntry{ID: "history-1", NovelID: novel.ID, ChapterID: chapter.ID, OpenedAt: time.Now().UTC()}
	if err := db.CreateHistory(entry); err != nil {
		t.Fatal(err)
	}
	closed := entry.OpenedAt.Add(time.Minute)
	if err := db.CloseHistory(entry.ID, closed, 60); err != nil {
		t.Fatal(err)
	}
	got, err := db.GetHistory(entry.ID)
	if err != nil || got.ClosedAt == nil || got.SessionSec != 60 {
		t.Fatalf("history: %#v, %v", got, err)
	}
}
func TestSettingsUpdate(t *testing.T) {
	db := testDB(t)
	settings, err := db.GetSettings()
	if err != nil || settings.LineWidth != 80 {
		t.Fatalf("defaults: %#v, %v", settings, err)
	}
	settings.LineWidth = 100
	settings.Theme = "light"
	settings.VimKeys = true
	if err := db.UpdateSettings(settings); err != nil {
		t.Fatal(err)
	}
	got, err := db.GetSettings()
	if err != nil || got.LineWidth != 100 || got.Theme != "light" || !got.VimKeys {
		t.Fatalf("update settings: %#v, %v", got, err)
	}
	var count int
	if err = db.SQL().QueryRow("SELECT count(*) FROM user_settings").Scan(&count); err != nil || count != 1 {
		t.Fatalf("settings rows: %d, %v", count, err)
	}
}

package source

import (
	"context"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/dharuncs/novel/internal/scraper"
)

func TestSuccessfulCallCleansWatchdog(t *testing.T) {
	plugin, err := LoadScript(`const source={id:"test",name:"Test",version:"1",baseURL:"https://example.com",language:"en",rateLimit:60,needsJS:false,search:function(){return []},novelInfo:function(){return {}},chapterList:function(){return []},chapterContent:function(){return ""}}`, scraper.NewClient())
	if err != nil {
		t.Fatal(err)
	}
	baseline := runtime.NumGoroutine()
	if _, err := plugin.Search(context.Background(), "x", 1); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	if current := runtime.NumGoroutine(); current > baseline+1 {
		t.Fatalf("watchdog goroutine leaked: baseline=%d current=%d", baseline, current)
	}
}

func TestJSExportToStructFieldMapping(t *testing.T) {
	script := `const source = {
		id: "test", name: "Test Source", version: "1.0", baseURL: "https://example.com", language: "en", rateLimit: 60, needsJS: false,
		novelInfo: function(url) {
			return {
				url: url,
				title: "Test Novel",
				author: "Test Author",
				coverURL: "https://example.com/cover.jpg",
				description: "A test description",
				status: "ongoing",
				tags: ["fantasy", "action"],
				totalChapters: 42
			};
		},
		chapterList: function(url) {
			return [
				{ url: "https://example.com/ch1", title: "Chapter 1", number: 1.0 },
				{ url: "https://example.com/ch1.5", title: "Chapter 1.5 Extra", number: 1.5 }
			];
		},
		search: function(q, page) {
			return [
				{ url: "https://example.com/found", title: "Found Title", author: "Found Author", coverURL: "https://example.com/img.png", status: "completed" }
			];
		},
		chapterContent: function(url) {
			return "Paragraph 1\n\nParagraph 2";
		}
	};`

	plugin, err := LoadScript(script, scraper.NewClient())
	if err != nil {
		t.Fatalf("failed to load test script: %v", err)
	}

	ctx := context.Background()

	// 1. Verify NovelInfo field unmarshaling
	novel, err := plugin.NovelInfo(ctx, "https://example.com/novel")
	if err != nil {
		t.Fatalf("NovelInfo error: %v", err)
	}
	if novel.URL != "https://example.com/novel" {
		t.Errorf("expected Novel.URL 'https://example.com/novel', got %q", novel.URL)
	}
	if novel.Title != "Test Novel" {
		t.Errorf("expected Novel.Title 'Test Novel', got %q", novel.Title)
	}
	if novel.Author != "Test Author" {
		t.Errorf("expected Novel.Author 'Test Author', got %q", novel.Author)
	}
	if novel.CoverURL != "https://example.com/cover.jpg" {
		t.Errorf("expected Novel.CoverURL 'https://example.com/cover.jpg', got %q", novel.CoverURL)
	}
	if novel.Description != "A test description" {
		t.Errorf("expected Novel.Description 'A test description', got %q", novel.Description)
	}
	if novel.Status != "ongoing" {
		t.Errorf("expected Novel.Status 'ongoing', got %q", novel.Status)
	}
	if len(novel.Tags) != 2 || novel.Tags[0] != "fantasy" || novel.Tags[1] != "action" {
		t.Errorf("expected Novel.Tags ['fantasy', 'action'], got %v", novel.Tags)
	}
	if novel.TotalChapters != 42 {
		t.Errorf("expected Novel.TotalChapters 42, got %d", novel.TotalChapters)
	}

	// 2. Verify ChapterList field unmarshaling
	chapters, err := plugin.ChapterList(ctx, "https://example.com/novel")
	if err != nil {
		t.Fatalf("ChapterList error: %v", err)
	}
	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(chapters))
	}
	if chapters[0].URL != "https://example.com/ch1" || chapters[0].Title != "Chapter 1" || chapters[0].Number != 1.0 {
		t.Errorf("chapter 0 field unmarshaling failed: %#v", chapters[0])
	}
	if chapters[1].URL != "https://example.com/ch1.5" || chapters[1].Title != "Chapter 1.5 Extra" || chapters[1].Number != 1.5 {
		t.Errorf("chapter 1 field unmarshaling failed: %#v", chapters[1])
	}

	// 3. Verify Search field unmarshaling
	results, err := plugin.Search(ctx, "test", 1)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 search result, got %d", len(results))
	}
	res := results[0]
	if res.URL != "https://example.com/found" || res.Title != "Found Title" || res.Author != "Found Author" || res.CoverURL != "https://example.com/img.png" || res.Status != "completed" {
		t.Errorf("search result unmarshaling failed: %#v", res)
	}

	// 4. Verify ChapterContent unmarshaling
	content, err := plugin.ChapterContent(ctx, "https://example.com/ch1")
	if err != nil {
		t.Fatalf("ChapterContent error: %v", err)
	}
	if content != "Paragraph 1\n\nParagraph 2" {
		t.Errorf("expected content 'Paragraph 1\\n\\nParagraph 2', got %q", content)
	}
}

func TestNovelfireSearchExactMatching(t *testing.T) {
	script, err := os.ReadFile("../../sources/novelfire.js")
	if err != nil {
		t.Fatalf("failed to read novelfire.js: %v", err)
	}
	plugin, err := LoadScript(string(script), scraper.NewClient())
	if err != nil {
		t.Fatalf("failed to load novelfire.js: %v", err)
	}

	results, err := plugin.Search(context.Background(), "lord of the mysteries", 1)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatalf("expected search results for 'lord of the mysteries', got 0")
	}

	if results[0].Title != "Lord of the Mysteries" {
		t.Errorf("expected first search result title 'Lord of the Mysteries', got %q", results[0].Title)
	}

	if results[0].URL != "https://novelfire.net/book/lord-of-the-mysteries" {
		t.Errorf("expected URL 'https://novelfire.net/book/lord-of-the-mysteries', got %q", results[0].URL)
	}
}

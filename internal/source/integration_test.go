//go:build integration

package source

import (
	"context"
	"os"
	"testing"

	"github.com/dharuncs/novel/internal/scraper"
)

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

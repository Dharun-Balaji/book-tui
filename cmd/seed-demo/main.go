package main

import (
	"fmt"
	"log"
	"time"

	"github.com/dharuncs/novel/internal/core"
	"github.com/dharuncs/novel/internal/source"
	"github.com/dharuncs/novel/internal/storage"
	"github.com/dharuncs/novel/internal/tui/chapterlist"
	"github.com/dharuncs/novel/internal/tui/reader"
	"github.com/dharuncs/novel/internal/tui/search"
)

func main() {
	db, err := storage.Open("demo_views.db")
	if err != nil {
		log.Fatalf("open storage: %v", err)
	}
	defer db.Close()

	now := time.Now().UTC()
	_ = db.UpsertSource(storage.Source{
		ID:        "novelfire",
		Name:      "Novel Fire",
		Version:   "1.0.0",
		BaseURL:   "https://novelfire.net",
		Language:  "en",
		RateLimit: 30,
	})

	novel := storage.Novel{
		ID:            "demo-novel-1",
		SourceID:      "novelfire",
		SourceURL:     "https://novelfire.net/book/lord-of-the-mysteries",
		Title:         "Lord of the Mysteries",
		Author:        "Cuttlefish That Loves Diving",
		CoverURL:      "https://novelfire.net/server-1/lord-of-the-mysteries.jpg",
		Description:   "In the waves of steam and machinery...",
		Status:        "completed",
		Tags:          []string{"fantasy", "mystery"},
		TotalChapters: 1432,
		InLibrary:     true,
		AddedAt:       now,
		UpdatedAt:     now,
	}
	_ = db.CreateNovel(novel)

	ch1 := storage.Chapter{
		ID:        "demo-ch-1",
		NovelID:   novel.ID,
		SourceURL: "https://novelfire.net/book/lord-of-the-mysteries/chapter-1",
		Number:    1.0,
		Title:     "Crimson",
		Content:   "Pain!\n\nHow painful!\n\nMy head hurts so much!",
		WordCount: 20,
		FetchedAt: &now,
		IsCached:  true,
	}
	ch2 := storage.Chapter{
		ID:        "demo-ch-2",
		NovelID:   novel.ID,
		SourceURL: "https://novelfire.net/book/lord-of-the-mysteries/chapter-2",
		Number:    2.0,
		Title:     "Situation",
		Content:   "Klein looked at the blood on the table.\n\nThe bullet hole on his temple was slowly healing.",
		WordCount: 30,
		FetchedAt: &now,
		IsCached:  true,
	}
	_ = db.CreateChapter(ch1)
	_ = db.CreateChapter(ch2)

	// 1. Render Search View
	fmt.Println("=== 1. SEARCH VIEW RENDER ===")
	searchM := search.New().SetSize(80, 10)
	searchM, _ = searchM.Update(search.SearchResultsMsg{
		Results: []source.SearchResult{
			{Title: "Lord of the Mysteries", Author: "Cuttlefish That Loves Diving", URL: "https://novelfire.net/book/lord-of-the-mysteries"},
			{Title: "Circle of Inevitability", Author: "Cuttlefish That Loves Diving", URL: "https://novelfire.net/book/circle-of-inevitability"},
		},
	})
	fmt.Println(searchM.View())

	// 2. Render Chapter List View
	fmt.Println("\n=== 2. CHAPTER LIST VIEW RENDER ===")
	chListM := chapterlist.New().SetSize(80, 10).SetChapters(core.Novel(novel), []core.Chapter{core.Chapter(ch1), core.Chapter(ch2)})
	fmt.Println(chListM.View())

	// 3. Render Reader Navigation (Chapter 2 after pressing 'n')
	fmt.Println("\n=== 3. READER NEXT CHAPTER RENDER (Chapter 2) ===")
	readerM := reader.New().SetChapter(novel.ID, core.Chapter(ch2), 0, 0, false, 80, 5).SetSize(80, 10)
	fmt.Println(readerM.View())
}

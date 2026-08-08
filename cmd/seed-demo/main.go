package main

import (
	"fmt"
	"log"
	"time"

	"github.com/dharuncs/novel/internal/storage"
	"github.com/dharuncs/novel/internal/tui/settings"
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

	_ = db.CreateNovel(storage.Novel{
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
	})

	fmt.Println("Seeded demo_views.db with Lord of the Mysteries!")

	darkSet := settings.New().SetSettings(storage.UserSettings{
		Theme:         "dark",
		LineWidth:     80,
		AutoSaveEvery: 5,
	}).SetSize(80, 5)
	fmt.Println(darkSet.View())
}

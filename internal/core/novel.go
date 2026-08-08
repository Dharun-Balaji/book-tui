package core

import (
	"fmt"
	"time"

	"github.com/dharuncs/novel/internal/source"
	"github.com/dharuncs/novel/internal/storage"
	"github.com/google/uuid"
)

// Type aliases for core domain models to avoid redundant mapping duplication.
type Novel = storage.Novel
type Chapter = storage.Chapter
type ReadingProgress = storage.ReadingProgress
type HistoryEntry = storage.HistoryEntry
type UserSettings = storage.UserSettings
type Source = storage.Source

// FromSourceNovel converts a scraped source.Novel into a storage.Novel domain entity.
func FromSourceNovel(sourceID string, sNovel source.Novel) Novel {
	now := time.Now().UTC()
	return Novel{
		ID:            uuid.New().String(),
		SourceID:      sourceID,
		SourceURL:     sNovel.URL,
		Title:         sNovel.Title,
		Author:        sNovel.Author,
		CoverURL:      sNovel.CoverURL,
		Description:   sNovel.Description,
		Status:        sNovel.Status,
		Tags:          sNovel.Tags,
		TotalChapters: sNovel.TotalChapters,
		InLibrary:     true,
		AddedAt:       now,
		UpdatedAt:     now,
	}
}

// FromSourceChapter converts a scraped source.Chapter into a storage.Chapter domain entity.
func FromSourceChapter(novelID string, sChapter source.Chapter) Chapter {
	return Chapter{
		ID:        uuid.New().String(),
		NovelID:   novelID,
		SourceURL: sChapter.URL,
		Number:    sChapter.Number,
		Title:     sChapter.Title,
		Content:   "",
		IsCached:  false,
	}
}

// CalculateProgressPct computes reading percentage for a given paragraph index.
func CalculateProgressPct(currentPara, totalParas int) float64 {
	if totalParas <= 0 {
		return 0.0
	}
	if currentPara >= totalParas {
		return 1.0
	}
	pct := float64(currentPara+1) / float64(totalParas)
	if pct < 0 {
		return 0
	}
	if pct > 1 {
		return 1
	}
	return pct
}

// FormatProgressString returns a readable progress string e.g. "Ch. 5 (45%)".
func FormatProgressString(chNum float64, pct float64) string {
	return fmt.Sprintf("Ch. %g (%.0f%%)", chNum, pct*100)
}

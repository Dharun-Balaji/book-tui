package core

import (
	"fmt"
	"time"

	"github.com/dharuncs/novel/internal/storage"
)

type ProgressManager struct {
	db *storage.DB
}

func NewProgressManager(db *storage.DB) *ProgressManager {
	return &ProgressManager{db: db}
}

// SaveReadingState persists reading position and updates cumulative statistics.
func (pm *ProgressManager) SaveReadingState(novelID, chapterID string, paragraphIdx, scrollOffset, totalParas int, additionalReadSec int64) error {
	chapter, err := pm.db.GetChapter(chapterID)
	if err != nil {
		return fmt.Errorf("get chapter for progress: %w", err)
	}

	currentProgress, err := pm.db.GetProgress(novelID)
	if err != nil {
		return fmt.Errorf("get current progress: %w", err)
	}

	pct := CalculateProgressPct(paragraphIdx, totalParas)
	now := time.Now().UTC()

	var totalSec int64
	var chaptersRead int

	if currentProgress != nil {
		totalSec = currentProgress.TotalReadSec + additionalReadSec
		chaptersRead = currentProgress.ChaptersRead
		// Increment chaptersRead if completed chapter (100%) and wasn't previously
		if pct >= 1.0 && currentProgress.ProgressPct < 1.0 {
			chaptersRead++
		}
	} else {
		totalSec = additionalReadSec
		if pct >= 1.0 {
			chaptersRead = 1
		}
	}

	progress := ReadingProgress{
		NovelID:      novelID,
		ChapterID:    chapterID,
		ChapterNum:   chapter.Number,
		ParagraphIdx: paragraphIdx,
		ScrollOffset: scrollOffset,
		ProgressPct:  pct,
		ChaptersRead: chaptersRead,
		TotalReadSec: totalSec,
		LastReadAt:   now,
	}

	return pm.db.SaveProgress(progress)
}

// GetResumeState resolves where to "Continue Reading" for a given novel.
// Returns (progress, targetChapter, error). If no progress exists, returns nil progress and chapter 1.
func (pm *ProgressManager) GetResumeState(novelID string) (*ReadingProgress, *Chapter, error) {
	progress, err := pm.db.GetProgress(novelID)
	if err != nil {
		return nil, nil, fmt.Errorf("query progress: %w", err)
	}

	chapters, err := pm.db.ListChapters(novelID)
	if err != nil {
		return nil, nil, fmt.Errorf("list chapters: %w", err)
	}
	if len(chapters) == 0 {
		return nil, nil, fmt.Errorf("novel %s has no chapters", novelID)
	}

	// No reading progress yet -> resume at first chapter
	if progress == nil {
		return nil, &chapters[0], nil
	}

	// Find the exact saved chapter
	for i, ch := range chapters {
		if ch.ID == progress.ChapterID {
			// If chapter was 100% completed and there is a next chapter, suggest next chapter
			if progress.ProgressPct >= 1.0 && i+1 < len(chapters) {
				nextCh := chapters[i+1]
				return progress, &nextCh, nil
			}
			return progress, &ch, nil
		}
	}

	// Fallback if saved chapter ID not found
	return progress, &chapters[0], nil
}

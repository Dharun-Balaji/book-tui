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

type ResumeState struct {
	Progress        *ReadingProgress `json:"progress"`
	Chapter         *Chapter         `json:"chapter"`
	IsNovelComplete bool             `json:"isNovelComplete"`
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
// Returns ResumeState containing progress, target chapter, and IsNovelComplete flag.
func (pm *ProgressManager) GetResumeState(novelID string) (ResumeState, error) {
	progress, err := pm.db.GetProgress(novelID)
	if err != nil {
		return ResumeState{}, fmt.Errorf("query progress: %w", err)
	}

	chapters, err := pm.db.ListChapters(novelID)
	if err != nil {
		return ResumeState{}, fmt.Errorf("list chapters: %w", err)
	}
	if len(chapters) == 0 {
		return ResumeState{}, fmt.Errorf("novel %s has no chapters", novelID)
	}

	// Case 1: No reading progress yet -> resume at first chapter (not complete)
	if progress == nil {
		return ResumeState{Progress: nil, Chapter: &chapters[0], IsNovelComplete: false}, nil
	}

	// Case 2: Saved chapter exists
	for i, ch := range chapters {
		if ch.ID == progress.ChapterID {
			if progress.ProgressPct >= 1.0 {
				if i+1 < len(chapters) {
					// Chapter complete, move to next chapter
					nextCh := chapters[i+1]
					return ResumeState{Progress: progress, Chapter: &nextCh, IsNovelComplete: false}, nil
				}
				// Last chapter complete -> novel complete!
				return ResumeState{Progress: progress, Chapter: &ch, IsNovelComplete: true}, nil
			}
			// In the middle of this chapter
			return ResumeState{Progress: progress, Chapter: &ch, IsNovelComplete: false}, nil
		}
	}

	// Fallback if saved chapter ID was removed or not found
	return ResumeState{Progress: progress, Chapter: &chapters[0], IsNovelComplete: false}, nil
}

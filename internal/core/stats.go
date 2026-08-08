package core

import (
	"fmt"
	"time"

	"github.com/dharuncs/novel/internal/storage"
	"github.com/google/uuid"
)

type StatsManager struct {
	db *storage.DB
}

func NewStatsManager(db *storage.DB) *StatsManager {
	return &StatsManager{db: db}
}

type OverallStats struct {
	TotalLibraryNovels int           `json:"totalLibraryNovels"`
	TotalChaptersRead  int           `json:"totalChaptersRead"`
	TotalReadingTime   time.Duration `json:"totalReadingTime"`
	TotalSessions      int           `json:"totalSessions"`
}

type NovelStats struct {
	NovelID        string         `json:"novelID"`
	ChaptersRead   int            `json:"chaptersRead"`
	ReadingTime    time.Duration  `json:"readingTime"`
	SessionCount   int            `json:"sessionCount"`
	SessionHistory []HistoryEntry `json:"sessionHistory"`
}

// StartReadingSession logs opening a chapter for a reading session.
func (sm *StatsManager) StartReadingSession(novelID, chapterID string) (HistoryEntry, error) {
	entry := HistoryEntry{
		ID:         uuid.New().String(),
		NovelID:    novelID,
		ChapterID:  chapterID,
		OpenedAt:   time.Now().UTC(),
		ClosedAt:   nil,
		SessionSec: 0,
	}
	if err := sm.db.CreateHistory(entry); err != nil {
		return HistoryEntry{}, fmt.Errorf("start session: %w", err)
	}
	return entry, nil
}

// EndReadingSession closes an active reading session and logs duration.
func (sm *StatsManager) EndReadingSession(historyID string) error {
	entry, err := sm.db.GetHistory(historyID)
	if err != nil {
		return fmt.Errorf("get history for end session: %w", err)
	}
	now := time.Now().UTC()
	duration := int64(now.Sub(entry.OpenedAt).Seconds())
	if duration < 0 {
		duration = 0
	}
	return sm.db.CloseHistory(historyID, now, duration)
}

// GetOverallStats aggregates reading metrics across the entire application.
func (sm *StatsManager) GetOverallStats() (OverallStats, error) {
	novels, err := sm.db.ListNovels(true)
	if err != nil {
		return OverallStats{}, err
	}

	totalChapters := 0
	totalSec := int64(0)

	for _, n := range novels {
		prog, err := sm.db.GetProgress(n.ID)
		if err != nil {
			return OverallStats{}, err
		}
		if prog != nil {
			totalChapters += prog.ChaptersRead
			totalSec += prog.TotalReadSec
		}
	}

	// Count total closed history sessions
	var sessionCount int
	err = sm.db.SQL().QueryRow("SELECT COUNT(*) FROM history WHERE closed_at IS NOT NULL").Scan(&sessionCount)
	if err != nil {
		return OverallStats{}, err
	}

	return OverallStats{
		TotalLibraryNovels: len(novels),
		TotalChaptersRead:  totalChapters,
		TotalReadingTime:   time.Duration(totalSec) * time.Second,
		TotalSessions:      sessionCount,
	}, nil
}

// GetNovelStats aggregates stats for a single novel.
func (sm *StatsManager) GetNovelStats(novelID string) (NovelStats, error) {
	history, err := sm.db.ListHistory(novelID)
	if err != nil {
		return NovelStats{}, err
	}

	prog, err := sm.db.GetProgress(novelID)
	if err != nil {
		return NovelStats{}, err
	}

	chaptersRead := 0
	totalSec := int64(0)
	if prog != nil {
		chaptersRead = prog.ChaptersRead
		totalSec = prog.TotalReadSec
	}

	return NovelStats{
		NovelID:        novelID,
		ChaptersRead:   chaptersRead,
		ReadingTime:    time.Duration(totalSec) * time.Second,
		SessionCount:   len(history),
		SessionHistory: history,
	}, nil
}

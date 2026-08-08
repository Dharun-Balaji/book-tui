package storage

import "time"

type Source struct {
	ID, Name, Version, BaseURL, Language string
	NeedsJS                              bool
	RateLimit                            int
}

type Novel struct {
	ID, SourceID, SourceURL, Title, Author, CoverURL, Description, Status string
	Tags                                                                  []string
	TotalChapters                                                         int
	InLibrary                                                             bool
	AddedAt, UpdatedAt                                                    time.Time
}

type Chapter struct {
	ID, NovelID, SourceURL, Title, Content string
	Number                                 float64
	WordCount                              int
	FetchedAt                              *time.Time
	IsCached                               bool
}

type ReadingProgress struct {
	NovelID, ChapterID                       string
	ChapterNum, ProgressPct                  float64
	ParagraphIdx, ScrollOffset, ChaptersRead int
	TotalReadSec                             int64
	LastReadAt                               time.Time
}

type HistoryEntry struct {
	ID, NovelID, ChapterID string
	OpenedAt               time.Time
	ClosedAt               *time.Time
	SessionSec             int64
}

type UserSettings struct {
	LineWidth, AutoSaveEvery int
	ScrollMode, Theme        string
	VimKeys, AutoSave        bool
}

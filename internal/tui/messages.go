package tui

import (
	"github.com/dharuncs/novel/internal/core"
)

// ViewState Enum defines the active screen/view in the TUI application.
type ViewState int

const (
	ViewLibrary ViewState = iota
	ViewReader
	ViewSearch
	ViewChapterList
	ViewSettings
)

// Navigation & View Switch Messages
type SwitchViewMsg struct{ State ViewState }
type BackMsg struct{}
type OpenNovelMsg struct{ NovelID string }

// OpenChapterMsg targets a specific chapter within a novel.
type OpenChapterMsg struct {
	NovelID   string
	ChapterID string
}

// Library Async Messages
type LoadLibraryMsg struct{}
type LibraryLoadedMsg struct{ Novels []core.Novel }
type AddNovelProgressMsg struct{ Status string }
type AddNovelSuccessMsg struct{ Novel core.Novel }
type AddNovelErrorMsg struct{ Err error }
type RemoveNovelMsg struct{ NovelID string }

// Reader Async Messages & Position Wiring
type FetchChapterMsg struct {
	NovelID   string
	ChapterID string
}

// ChapterLoadedMsg carries chapter content AND resume position data for instant seeking upon load.
type ChapterLoadedMsg struct {
	Chapter         core.Chapter
	ParagraphIdx    int
	ScrollOffset    int
	IsNovelComplete bool
}

type ChapterErrorMsg struct{ Err error }

// SaveProgressMsg persists reading position and time spent.
type SaveProgressMsg struct {
	NovelID           string
	ChapterID         string
	ParagraphIdx      int
	ScrollOffset      int
	TotalParas        int
	AdditionalReadSec int64
}

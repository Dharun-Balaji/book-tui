package reader

type SaveProgressMsg struct {
	NovelID           string
	ChapterID         string
	ParagraphIdx      int
	ScrollOffset      int
	TotalParas        int
	AdditionalReadSec int64
}

type SwitchToLibraryMsg struct{}

type NextChapterMsg struct {
	NovelID           string
	CurrentChapterNum float64
}

type PrevChapterMsg struct {
	NovelID           string
	CurrentChapterNum float64
}

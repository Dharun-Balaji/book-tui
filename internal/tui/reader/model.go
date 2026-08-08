package reader

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dharuncs/novel/internal/core"
)

type Model struct {
	novelID           string
	chapter           core.Chapter
	paragraphs        []string
	paragraphIdx      int
	scrollOffset      int
	scrolledSinceSave int
	autoSaveEvery     int
	lineWidth         int
	width             int
	height            int
	viewport          viewport.Model
	isNovelComplete   bool
	statusMsg         string
	readStartTime     time.Time
	ready             bool
}

func New() Model {
	vp := viewport.New(0, 0)
	return Model{
		viewport:      vp,
		lineWidth:     80,
		autoSaveEvery: 5,
		readStartTime: time.Now().UTC(),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) SetChapter(
	novelID string,
	chapter core.Chapter,
	pIdx, sOffset int,
	isComplete bool,
	lineWidth, autoSaveEvery int,
) Model {
	m.novelID = novelID
	m.chapter = chapter
	m.isNovelComplete = isComplete

	if lineWidth <= 0 {
		lineWidth = 80
	}
	m.lineWidth = lineWidth

	if autoSaveEvery <= 0 {
		autoSaveEvery = 5
	}
	m.autoSaveEvery = autoSaveEvery

	m.readStartTime = time.Now().UTC()
	m.scrolledSinceSave = 0

	// Split content into raw paragraphs
	rawParas := strings.Split(chapter.Content, "\n\n")
	m.paragraphs = make([]string, 0, len(rawParas))
	for _, p := range rawParas {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			m.paragraphs = append(m.paragraphs, trimmed)
		}
	}

	if pIdx < 0 {
		pIdx = 0
	}
	if pIdx >= len(m.paragraphs) && len(m.paragraphs) > 0 {
		pIdx = len(m.paragraphs) - 1
	}
	m.paragraphIdx = pIdx
	m.scrollOffset = sOffset

	m.ready = true
	if m.width > 0 && m.height > 0 {
		m = m.SetSize(m.width, m.height)
	} else {
		m = m.renderViewport()
		m = m.seekToParagraph(m.paragraphIdx, m.scrollOffset)
	}

	return m
}

func (m Model) GetCurrentState() (novelID, chapterID string, pIdx, sOffset, totalParas int, readSec int64) {
	readSec = int64(time.Since(m.readStartTime).Seconds())
	if readSec < 0 {
		readSec = 0
	}
	return m.novelID, m.chapter.ID, m.paragraphIdx, m.scrollOffset, len(m.paragraphs), readSec
}

func (m Model) SetSize(width, height int) Model {
	m.width = width
	m.height = height
	vpHeight := height - 4 // Reserve 2 lines header, 2 lines footer
	if vpHeight < 1 {
		vpHeight = 1
	}
	m.viewport.Width = width
	m.viewport.Height = vpHeight

	if m.ready {
		// Paragraph-anchored reflow on resize: re-render & re-seek to paragraphIdx
		m = m.renderViewport()
		m = m.seekToParagraph(m.paragraphIdx, m.scrollOffset)
	}

	return m
}

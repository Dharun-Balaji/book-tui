package reader

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			novelID, chID, pIdx, sOffset, totalP, readSec := m.GetCurrentState()
			saveCmd := func() tea.Msg {
				return SaveProgressMsg{
					NovelID:           novelID,
					ChapterID:         chID,
					ParagraphIdx:      pIdx,
					ScrollOffset:      sOffset,
					TotalParas:        totalP,
					AdditionalReadSec: readSec,
				}
			}
			backCmd := func() tea.Msg {
				return SwitchToLibraryMsg{}
			}
			return m, tea.Batch(saveCmd, backCmd)

		case "n":
			novelID, chID, pIdx, sOffset, totalP, readSec := m.GetCurrentState()
			saveCmd := func() tea.Msg {
				return SaveProgressMsg{
					NovelID:           novelID,
					ChapterID:         chID,
					ParagraphIdx:      pIdx,
					ScrollOffset:      sOffset,
					TotalParas:        totalP,
					AdditionalReadSec: readSec,
				}
			}
			nextCmd := func() tea.Msg {
				return NextChapterMsg{
					NovelID:           novelID,
					CurrentChapterNum: m.chapter.Number,
				}
			}
			return m, tea.Batch(saveCmd, nextCmd)

		case "p":
			novelID, chID, pIdx, sOffset, totalP, readSec := m.GetCurrentState()
			saveCmd := func() tea.Msg {
				return SaveProgressMsg{
					NovelID:           novelID,
					ChapterID:         chID,
					ParagraphIdx:      pIdx,
					ScrollOffset:      sOffset,
					TotalParas:        totalP,
					AdditionalReadSec: readSec,
				}
			}
			prevCmd := func() tea.Msg {
				return PrevChapterMsg{
					NovelID:           novelID,
					CurrentChapterNum: m.chapter.Number,
				}
			}
			return m, tea.Batch(saveCmd, prevCmd)

		case "b":
			m.statusMsg = "Bookmarks not yet implemented in storage"

		case "g":
			m.paragraphIdx = 0
			m.scrollOffset = 0
			m = m.seekToParagraph(0, 0)

		case "G":
			if len(m.paragraphs) > 0 {
				m.paragraphIdx = len(m.paragraphs) - 1
				m.scrollOffset = 0
				m = m.seekToParagraph(m.paragraphIdx, 0)
			}

		case "j", "down":
			m.viewport.LineDown(1)
			m = m.updatePositionFromYOffset()
			if m.scrolledSinceSave >= m.autoSaveEvery {
				m.scrolledSinceSave = 0
				cmds = append(cmds, m.buildSaveCmd())
			}

		case "k", "up":
			m.viewport.LineUp(1)
			m = m.updatePositionFromYOffset()
		}
	}

	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

func (m Model) buildSaveCmd() tea.Cmd {
	novelID, chID, pIdx, sOffset, totalP, readSec := m.GetCurrentState()
	return func() tea.Msg {
		return SaveProgressMsg{
			NovelID:           novelID,
			ChapterID:         chID,
			ParagraphIdx:      pIdx,
			ScrollOffset:      sOffset,
			TotalParas:        totalP,
			AdditionalReadSec: readSec,
		}
	}
}

func (m Model) updatePositionFromYOffset() Model {
	if len(m.paragraphs) == 0 {
		return m
	}
	y := m.viewport.YOffset
	accum := 0
	oldPIdx := m.paragraphIdx

	for i, p := range m.paragraphs {
		lines := wrapLine(p, m.lineWidth)
		pLineCount := len(lines) + 1 // +1 for blank spacing line
		if accum+pLineCount > y {
			m.paragraphIdx = i
			m.scrollOffset = y - accum
			break
		}
		accum += pLineCount
	}

	if m.paragraphIdx != oldPIdx {
		m.scrolledSinceSave++
	}
	return m
}

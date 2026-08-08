package reader

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/clipperhouse/displaywidth"
	"github.com/dharuncs/novel/internal/core"
)

func wrapLine(text string, width int) []string {
	if width <= 0 {
		width = 80
	}
	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var currentLine string
	currentWidth := 0

	for _, word := range words {
		wWidth := displaywidth.String(word)
		if currentWidth == 0 {
			currentLine = word
			currentWidth = wWidth
		} else if currentWidth+1+wWidth <= width {
			currentLine += " " + word
			currentWidth += 1 + wWidth
		} else {
			lines = append(lines, currentLine)
			currentLine = word
			currentWidth = wWidth
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}
	return lines
}

func (m Model) renderViewport() Model {
	if len(m.paragraphs) == 0 {
		m.viewport.SetContent("Empty chapter content.")
		return m
	}

	var renderedParagraphs []string
	for i, p := range m.paragraphs {
		wrappedLines := wrapLine(p, m.lineWidth)
		// Mark paragraph header tag e.g. [P1] for tracking line offsets
		pText := strings.Join(wrappedLines, "\n")
		if i < len(m.paragraphs)-1 {
			pText += "\n" // Blank line between paragraphs
		}
		renderedParagraphs = append(renderedParagraphs, pText)
	}

	m.viewport.SetContent(strings.Join(renderedParagraphs, "\n"))
	return m
}

func (m Model) seekToParagraph(pIdx, sOffset int) Model {
	if len(m.paragraphs) == 0 {
		return m
	}
	lineOffset := 0
	for i := 0; i < pIdx && i < len(m.paragraphs); i++ {
		lines := wrapLine(m.paragraphs[i], m.lineWidth)
		lineOffset += len(lines) + 1 // +1 for blank spacing line
	}
	lineOffset += sOffset
	m.viewport.SetYOffset(lineOffset)
	return m
}

func (m Model) View() string {
	if !m.ready {
		return "Loading chapter..."
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Padding(0, 1).
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(lipgloss.Color("238"))

	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(0, 1).
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(lipgloss.Color("238"))

	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	title := fmt.Sprintf("Chapter %g — %s", m.chapter.Number, m.chapter.Title)
	if m.chapter.Title == "" {
		title = fmt.Sprintf("Chapter %g", m.chapter.Number)
	}
	header := headerStyle.Render(title)

	totalP := len(m.paragraphs)
	currentP := m.paragraphIdx + 1
	pct := core.CalculateProgressPct(m.paragraphIdx, totalP)

	statusStr := fmt.Sprintf("Para %d/%d (%.0f%%)", currentP, totalP, pct*100)
	if m.isNovelComplete {
		statusStr += " • [NOVEL COMPLETED]"
	}
	if m.statusMsg != "" {
		statusStr += " • " + statusStyle.Render(m.statusMsg)
	}

	controls := "j/k: scroll • g/G: top/bot • n/p: next/prev • b: bookmark • q: back"
	footer := footerStyle.Render(statusStr + "\n" + controls)

	return lipgloss.JoinVertical(lipgloss.Left, header, m.viewport.View(), footer)
}

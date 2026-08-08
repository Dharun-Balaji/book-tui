package core

import (
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/dharuncs/novel/internal/source"
	"github.com/dharuncs/novel/internal/storage"
	"github.com/google/uuid"
)

var (
	regexpBold    = regexp.MustCompile(`(?i)<(?:b|strong)>(.*?)</(?:b|strong)>`)
	regexpItalic  = regexp.MustCompile(`(?i)<(?:i|em)>(.*?)</(?:i|em)>`)
	regexpHTMLTag = regexp.MustCompile(`<[^>]*>`)
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

// CleanContent strips residual HTML tags, unescapes HTML entities, and normalizes line breaks.
func CleanContent(rawHTML string) string {
	// 1. Unescape HTML entities (&quot;, &amp;, &lt;, &gt;, &#39;, &nbsp;, etc.)
	str := html.UnescapeString(rawHTML)
	str = strings.ReplaceAll(str, "\u00a0", " ") // Non-breaking space

	// 2. Replace formatting HTML tags with simple markdown equivalents
	str = strings.ReplaceAll(str, "<br>", "\n")
	str = strings.ReplaceAll(str, "<br/>", "\n")
	str = strings.ReplaceAll(str, "<br />", "\n")
	str = regexpBold.ReplaceAllString(str, "**$1**")
	str = regexpItalic.ReplaceAllString(str, "*$1*")

	// 3. Strip any remaining HTML tags
	str = regexpHTMLTag.ReplaceAllString(str, "")

	// 4. Normalize multiple consecutive blank lines to double newlines (\n\n)
	lines := strings.Split(str, "\n")
	var cleaned []string
	blank := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if !blank && len(cleaned) > 0 {
				cleaned = append(cleaned, "")
				blank = true
			}
		} else {
			cleaned = append(cleaned, trimmed)
			blank = false
		}
	}
	return strings.Join(cleaned, "\n")
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

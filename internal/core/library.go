package core

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dharuncs/novel/internal/source"
	"github.com/dharuncs/novel/internal/storage"
)

type LibraryManager struct {
	db *storage.DB
}

func NewLibraryManager(db *storage.DB) *LibraryManager {
	return &LibraryManager{db: db}
}

// IsNovelInLibrary checks if a novel from a specific source is already tracked in the library.
func (lm *LibraryManager) IsNovelInLibrary(sourceID, sourceURL string) (bool, *Novel, error) {
	novels, err := lm.db.ListNovels(false)
	if err != nil {
		return false, nil, err
	}
	for _, n := range novels {
		if n.SourceID == sourceID && n.SourceURL == sourceURL {
			return n.InLibrary, &n, nil
		}
	}
	return false, nil, nil
}

type ProgressCallback func(status string)

// AddNovel fetches novel info & chapter list from a source plugin, persists it to storage, and flags it in library.
func (lm *LibraryManager) AddNovel(ctx context.Context, plugin *source.Plugin, sourceNovelURL string, onProgress ProgressCallback) (Novel, error) {
	report := func(msg string) {
		if onProgress != nil {
			onProgress(msg)
		}
	}

	plugin.OnProgress = func(targetURL string) {
		if strings.Contains(targetURL, "page=") {
			parts := strings.Split(targetURL, "page=")
			if len(parts) > 1 {
				page := strings.Split(parts[1], "&")[0]
				report(fmt.Sprintf("Fetching chapter list page %s...", page))
				return
			}
		}
		report("Fetching metadata from source...")
	}
	defer func() { plugin.OnProgress = nil }()

	report("Updating source metadata...")

	// Upsert source metadata to DB first
	meta := plugin.Metadata
	if err := lm.db.UpsertSource(storage.Source{
		ID:        meta.ID,
		Name:      meta.Name,
		Version:   meta.Version,
		BaseURL:   meta.BaseURL,
		Language:  meta.Language,
		NeedsJS:   meta.NeedsJS,
		RateLimit: meta.RateLimit,
	}); err != nil {
		return Novel{}, fmt.Errorf("upsert source metadata: %w", err)
	}

	// Check if already in DB
	exists, existing, err := lm.IsNovelInLibrary(meta.ID, sourceNovelURL)
	if err != nil {
		return Novel{}, err
	}
	if exists && existing != nil {
		return *existing, nil
	}

	report("Fetching novel metadata...")
	// Fetch novel metadata from plugin
	sNovel, err := plugin.NovelInfo(ctx, sourceNovelURL)
	if err != nil {
		return Novel{}, fmt.Errorf("fetch novel info: %w", err)
	}

	report("Fetching chapter list...")
	// Fetch chapter list from plugin
	sChapters, err := plugin.ChapterList(ctx, sourceNovelURL)
	if err != nil {
		return Novel{}, fmt.Errorf("fetch chapter list: %w", err)
	}

	report(fmt.Sprintf("Saving novel and %d chapters to library...", len(sChapters)))

	novel := FromSourceNovel(meta.ID, sNovel)
	novel.TotalChapters = len(sChapters)

	if existing != nil {
		// Update existing record to set in_library = true
		novel.ID = existing.ID
		if err := lm.db.UpdateNovel(novel); err != nil {
			return Novel{}, fmt.Errorf("update novel: %w", err)
		}
	} else {
		// Create new novel entry
		if err := lm.db.CreateNovel(novel); err != nil {
			return Novel{}, fmt.Errorf("create novel: %w", err)
		}
	}

	// Persist chapter stubs
	for _, sCh := range sChapters {
		ch := FromSourceChapter(novel.ID, sCh)
		if err := lm.db.CreateChapter(ch); err != nil {
			// Ignore duplicate chapter error if updating
			if !strings.Contains(err.Error(), "UNIQUE constraint failed") {
				return Novel{}, fmt.Errorf("create chapter %.1f: %w", sCh.Number, err)
			}
		}
	}

	return novel, nil
}

// RemoveNovel marks a novel as not in library (or deletes if completely untracked).
func (lm *LibraryManager) RemoveNovel(id string) error {
	novel, err := lm.db.GetNovel(id)
	if err != nil {
		return err
	}
	novel.InLibrary = false
	novel.UpdatedAt = time.Now().UTC()
	return lm.db.UpdateNovel(novel)
}

// ListLibrary returns all novels marked in_library = true.
func (lm *LibraryManager) ListLibrary() ([]Novel, error) {
	return lm.db.ListNovels(true)
}

// GetNovel retrieves a single novel by ID.
func (lm *LibraryManager) GetNovel(id string) (Novel, error) {
	return lm.db.GetNovel(id)
}

// ListChapters returns all chapters for a given novel ordered by chapter number.
func (lm *LibraryManager) ListChapters(novelID string) ([]Chapter, error) {
	return lm.db.ListChapters(novelID)
}

// GetChapterContent returns chapter content. Fetches from plugin & caches to DB if not already cached.
func (lm *LibraryManager) GetChapterContent(ctx context.Context, plugin *source.Plugin, chapterID string) (Chapter, error) {
	chapter, err := lm.db.GetChapter(chapterID)
	if err != nil {
		return Chapter{}, err
	}

	// If content is cached, return directly
	if chapter.IsCached && chapter.Content != "" {
		return chapter, nil
	}

	// Fetch chapter content from plugin
	rawContent, err := plugin.ChapterContent(ctx, chapter.SourceURL)
	if err != nil {
		return Chapter{}, fmt.Errorf("fetch chapter content: %w", err)
	}

	content := CleanContent(rawContent)
	now := time.Now().UTC()
	words := len(strings.Fields(content))

	chapter.Content = content
	chapter.WordCount = words
	chapter.FetchedAt = &now
	chapter.IsCached = true

	if err := lm.db.UpdateChapter(chapter); err != nil {
		return Chapter{}, fmt.Errorf("cache chapter content: %w", err)
	}

	return chapter, nil
}

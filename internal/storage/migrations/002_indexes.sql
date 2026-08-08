-- Migration 002: Bookmarks note length enforcement + history index improvement
-- Kept separate from 001 so we can audit the evolution of the schema.

-- SQLite doesn't support ALTER COLUMN, so structural changes that need a new
-- constraint go via a shadow table + copy + rename pattern. For now this
-- migration just adds two useful indexes that weren't in the initial schema.

UPDATE schema_version SET version = 2;

-- Faster "what did I read recently across all novels?" query (used by the
-- "Last Read" list on the library home screen).
CREATE INDEX IF NOT EXISTS idx_history_recent
    ON history(closed_at DESC)
    WHERE closed_at IS NOT NULL;

-- Faster per-chapter bookmark lookup (used when opening a chapter to show
-- the bookmark indicator on paragraphs).
CREATE INDEX IF NOT EXISTS idx_bookmarks_chapter
    ON bookmarks(chapter_id);

-- Migration 001: Initial schema
-- Applied once on first launch. All subsequent changes go in 002_, 003_, etc.

PRAGMA journal_mode = WAL;   -- concurrent reads while a write is in progress
PRAGMA foreign_keys = ON;    -- enforce FK constraints at the SQLite level

-- ─── schema_version ─────────────────────────────────────────────────────────
-- Simple monotonic counter. The Go migration runner checks this on startup
-- and applies any migration files whose index > current version.
CREATE TABLE IF NOT EXISTS schema_version (
    version     INTEGER NOT NULL
);
INSERT INTO schema_version (version) VALUES (1);

-- ─── sources ─────────────────────────────────────────────────────────────────
-- One row per loaded JS plugin. Populated at startup from the sources/ dir.
-- Kept in DB so we can store per-source state (last_checked, disabled flag).
CREATE TABLE IF NOT EXISTS sources (
    id              TEXT PRIMARY KEY,           -- e.g. "royalroad"
    name            TEXT NOT NULL,              -- "Royal Road"
    version         TEXT NOT NULL,              -- semver from plugin header
    base_url        TEXT NOT NULL,
    language        TEXT NOT NULL DEFAULT 'en',
    needs_js        INTEGER NOT NULL DEFAULT 0, -- boolean: 0=false, 1=true
    rate_limit      INTEGER NOT NULL DEFAULT 60 -- requests per minute
);

-- ─── novels ──────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS novels (
    id              TEXT PRIMARY KEY,           -- UUID v4
    source_id       TEXT NOT NULL REFERENCES sources(id) ON DELETE RESTRICT,
    source_url      TEXT NOT NULL,              -- canonical URL on the source site
    title           TEXT NOT NULL,
    author          TEXT NOT NULL DEFAULT '',
    cover_url       TEXT NOT NULL DEFAULT '',   -- remote URL; cached separately
    description     TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'unknown'
                        CHECK(status IN ('ongoing','completed','hiatus','unknown')),
    total_chapters  INTEGER NOT NULL DEFAULT 0,
    in_library      INTEGER NOT NULL DEFAULT 0, -- boolean: 1 = user added to library
    added_at        TEXT NOT NULL,              -- ISO-8601 UTC
    updated_at      TEXT NOT NULL,              -- last metadata refresh

    -- Enforce that the same novel URL from the same source is never duplicated
    UNIQUE(source_id, source_url)
);

CREATE INDEX IF NOT EXISTS idx_novels_source    ON novels(source_id);
CREATE INDEX IF NOT EXISTS idx_novels_library   ON novels(in_library);
CREATE INDEX IF NOT EXISTS idx_novels_title     ON novels(title COLLATE NOCASE);

-- ─── novel_tags ──────────────────────────────────────────────────────────────
-- Tags normalised into a separate table to avoid storing comma-separated blobs.
CREATE TABLE IF NOT EXISTS novel_tags (
    novel_id    TEXT NOT NULL REFERENCES novels(id) ON DELETE CASCADE,
    tag         TEXT NOT NULL,
    PRIMARY KEY (novel_id, tag)
);

-- ─── chapters ────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS chapters (
    id          TEXT PRIMARY KEY,               -- UUID v4
    novel_id    TEXT NOT NULL REFERENCES novels(id) ON DELETE CASCADE,
    source_url  TEXT NOT NULL,
    number      REAL NOT NULL,                  -- float: handles "ch 1.5" extras
    title       TEXT NOT NULL DEFAULT '',
    content     TEXT NOT NULL DEFAULT '',       -- cleaned plain text; empty until fetched
    word_count  INTEGER NOT NULL DEFAULT 0,
    fetched_at  TEXT,                           -- NULL = not yet cached
    is_cached   INTEGER NOT NULL DEFAULT 0,     -- boolean

    UNIQUE(novel_id, number),
    UNIQUE(novel_id, source_url)
);

CREATE INDEX IF NOT EXISTS idx_chapters_novel   ON chapters(novel_id);
CREATE INDEX IF NOT EXISTS idx_chapters_num     ON chapters(novel_id, number);
CREATE INDEX IF NOT EXISTS idx_chapters_cached  ON chapters(novel_id, is_cached);

-- ─── reading_progress ────────────────────────────────────────────────────────
-- One row per novel. Upserted (INSERT OR REPLACE) on every save.
CREATE TABLE IF NOT EXISTS reading_progress (
    novel_id        TEXT PRIMARY KEY REFERENCES novels(id) ON DELETE CASCADE,
    chapter_id      TEXT NOT NULL REFERENCES chapters(id) ON DELETE RESTRICT,
    chapter_num     REAL NOT NULL,
    paragraph_idx   INTEGER NOT NULL DEFAULT 0, -- viewport-top paragraph (resize-stable)
    scroll_offset   INTEGER NOT NULL DEFAULT 0, -- fine offset within that paragraph
    progress_pct    REAL NOT NULL DEFAULT 0.0,  -- 0.0–1.0 within chapter
    chapters_read   INTEGER NOT NULL DEFAULT 0, -- total chapters completed
    total_read_sec  INTEGER NOT NULL DEFAULT 0, -- cumulative seconds across all sessions
    last_read_at    TEXT NOT NULL               -- ISO-8601 UTC
);

-- ─── bookmarks ───────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS bookmarks (
    id              TEXT PRIMARY KEY,           -- UUID v4
    novel_id        TEXT NOT NULL REFERENCES novels(id) ON DELETE CASCADE,
    chapter_id      TEXT NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
    paragraph_idx   INTEGER NOT NULL,
    note            TEXT NOT NULL DEFAULT '',
    created_at      TEXT NOT NULL               -- ISO-8601 UTC
);

CREATE INDEX IF NOT EXISTS idx_bookmarks_novel  ON bookmarks(novel_id);

-- ─── history ─────────────────────────────────────────────────────────────────
-- Append-only log: one row per reading session (open → close).
CREATE TABLE IF NOT EXISTS history (
    id          TEXT PRIMARY KEY,               -- UUID v4
    novel_id    TEXT NOT NULL REFERENCES novels(id) ON DELETE CASCADE,
    chapter_id  TEXT NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
    opened_at   TEXT NOT NULL,                  -- ISO-8601 UTC
    closed_at   TEXT,                           -- NULL = session still open
    session_sec INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_history_novel    ON history(novel_id);
CREATE INDEX IF NOT EXISTS idx_history_opened   ON history(opened_at);

-- ─── user_settings ───────────────────────────────────────────────────────────
-- Single-row settings table. Always seeded with defaults on first run.
-- Use UPDATE; never INSERT a second row.
CREATE TABLE IF NOT EXISTS user_settings (
    id              INTEGER PRIMARY KEY CHECK(id = 1), -- enforces single-row
    line_width      INTEGER NOT NULL DEFAULT 80,
    scroll_mode     TEXT    NOT NULL DEFAULT 'line'
                        CHECK(scroll_mode IN ('line','page')),
    theme           TEXT    NOT NULL DEFAULT 'dark',
    vim_keys        INTEGER NOT NULL DEFAULT 0,         -- boolean
    auto_save       INTEGER NOT NULL DEFAULT 1,         -- boolean
    auto_save_every INTEGER NOT NULL DEFAULT 5          -- paragraphs
);

-- Seed defaults on migration (will be no-op if row already exists)
INSERT OR IGNORE INTO user_settings (id) VALUES (1);

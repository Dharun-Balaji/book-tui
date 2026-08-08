# novel — Project Context

> Living document. Updated as decisions are made, findings are verified, or scope changes.  
> Last updated: 2026-08-08

---

## Project Identity

| Field | Value |
|---|---|
| **Project name** | `novel` (binary name TBD — see open questions) |
| **Goal** | Feature-rich, lightweight, fast terminal-based novel/light-novel reader |
| **Inspiration** | LNReader (Android) — source-plugin system, library management, reading UX |
| **Target user** | Single local user; SSH-compatible terminal |
| **Current phase** | Steps 1–2 complete; Novelfire diagnostic complete; core layer is next |

---

## Confirmed Technical Decisions

### Stack
| Layer | Choice | Reason |
|---|---|---|
| Language | **Go** | Goroutine concurrency, ~5ms startup, single binary distribution |
| TUI framework | **Bubble Tea** + Lip Gloss + Bubbles | Elm architecture, mature ecosystem |
| Plugin engine | **goja** (JS in Go) | LNReader plugins are JS; can port directly; ES5.1+ES6 subset; no npm |
| HTML parsing (Go bridge) | **goquery** | jQuery-like, used inside the Go `fetch()` bridge exposed to plugins |
| Storage | **modernc/sqlite** (pure Go) | No cgo, no gcc, true cross-compilation; relational schema fits the data |
| Config file | **TOML** via BurntSushi/toml | Human-editable, simple |
| CLI routing | **cobra** | Subcommands: `novel`, `novel source test <id>`, etc. |
| Fuzzy search | **sahilm/fuzzy** | Library filter + search results |

### Storage Format
- **SQLite** chosen over JSON/TOML (no efficient queries, full rewrite on save) and bbolt (no SQL, harder cross-entity queries)
- Driver: `modernc.org/sqlite` — pure Go, same `database/sql` API as `mattn/go-sqlite3` but no cgo/gcc dependency → enables `GOARCH=arm64 GOOS=darwin go build` without a cross-compiler
- Migrations: hand-rolled, sequential numbered SQL files in `internal/storage/migrations/`; `schema_version` table tracks applied version

### Plugin System
- **Model**: JS files dropped in `sources/` directory — no recompile needed, like LNReader
- **Engine**: `goja` isolated runtime per plugin; sandboxed (no filesystem, no `require()`, no raw network)
- **Go bridge**: plugins call `fetch(url)` → Go's `scraper/client.go` (rate limiter, retry, header injection, caching) → HTML string back to JS
- **Domain allowlist**: `fetch()` enforces plugin can only fetch from its declared `baseURL`
- **Timeout**: 30s idle timeout reset after successful fetches plus a 5-minute outer ceiling; errors distinguish idle from outer timeout
- **Plugin spec**: each `.js` file exports a `source` object with: `id`, `name`, `version`, `baseURL`, `language`, `rateLimit`, `needsJS`, and methods `search()`, `novelInfo()`, `chapterList()`, `chapterContent()`
- **Test subcommand**: `novel source test <id>` runs a fixed URL through the plugin and validates output schema

### Reading Position Persistence
- Stored as **`paragraph_idx`** (not line number) — resize-stable because terminal reflow changes line counts but not paragraph counts
- `scroll_offset` stored as a fine-position hint within that paragraph
- Auto-save trigger: every 5 paragraphs scrolled (configurable) + on navigation away + on `tea.Quit`
- Restore flow: query `reading_progress` → load chapter → render paragraph slice → seek viewport to `paragraph_idx`

### Distribution
- **Binary + `sources/` directory** alongside it
- Installable via `go install`; `sources/` ships beside the binary

### Sync / Multi-device
- **Local-first** for now
- Storage layer designed so future sync is not painful: export/import DB file, or Litestream replication hook

---

## Sources List

| Plugin file | Site | `needsJS` | Notes |
|---|---|---|---|
| `royalroad.js` | royalroad.com | `false` | Second plugin; not started |
| `novelbin.js` | novelbin.com | `false` | Phase 2 |
| `lightnovelpub.js` | lightnovelpub.com | `false` | Phase 2 |
| `novelfire.js` | novelfire.net | **`false`** | **MVP — first plugin; implemented** |

### novelfire.net — Verified Finding (2026-08-08)
- **Prior assumption (wrong)**: flagged `needsJS: true` based on a 403 response from a headerless `read_url_content` fetch
- **Root cause of 403**: Cloudflare bot-detection triggered by absence of `User-Agent`, `Accept`, `Sec-Fetch-*` headers — not a JS challenge
- **Verified**: `curl` with full browser headers (`User-Agent: Chrome/126`, `Accept`, `Accept-Language`, `Referer`, `Sec-Fetch-*`) → **HTTP 200**, 9.9KB of fully server-rendered HTML with chapter links (`/book/.../chapter-1` through `chapter-N`) present directly in markup
- **Cloudflare beacon**: page includes a passive `__cf_chl` telemetry `<script>` tag — this is **not** a JS challenge gate
- **Conclusion**: `novelfire.js` sets `needsJS: false`, `rateLimit: 30`. The shared `client.go` global browser-header defaults handle it entirely. No `chromedp` needed for this source
- **Implication**: `chromedp` headless fallback stays in Phase 3 (not pulled forward to Phase 2)
- **Live diagnostic**: the provided Lord of the Mysteries URL returns HTTP 200; pages 1–14 each return 100 chapters. The external command window ended before page 15.

---

## Data Models (finalized for schema review)

### `Novel`
```go
type Novel struct {
    ID            string    // UUID v4
    SourceID      string    // FK → sources.id
    SourceURL     string    // canonical URL on source site
    Title         string
    Author        string
    CoverURL      string
    Description   string
    Tags          []string  // stored in novel_tags junction table
    Status        string    // "ongoing"|"completed"|"hiatus"|"unknown"
    TotalChapters int
    InLibrary     bool      // false = browsed/searched but not added
    AddedAt       time.Time
    UpdatedAt     time.Time
}
```

### `Chapter`
```go
type Chapter struct {
    ID        string
    NovelID   string
    SourceURL string
    Number    float64   // float: handles "ch 1.5" side-stories
    Title     string
    Content   string    // cleaned plain text; empty until fetched (is_cached=false)
    WordCount int
    FetchedAt *time.Time
    IsCached  bool
}
```

### `ReadingProgress`
```go
type ReadingProgress struct {
    NovelID      string
    ChapterID    string
    ChapterNum   float64
    ParagraphIdx int       // viewport-top paragraph index (resize-stable)
    ScrollOffset int       // fine offset within paragraph
    ProgressPct  float64   // 0.0–1.0 within chapter
    ChaptersRead int
    TotalReadSec int64
    LastReadAt   time.Time
}
```

### `Bookmark`
```go
type Bookmark struct {
    ID           string
    NovelID      string
    ChapterID    string
    ParagraphIdx int
    Note         string
    CreatedAt    time.Time
}
```

### `HistoryEntry`
```go
type HistoryEntry struct {
    ID         string
    NovelID    string
    ChapterID  string
    OpenedAt   time.Time
    ClosedAt   *time.Time // nil = session still open
    SessionSec int64
}
```

### `UserSettings`
```go
type UserSettings struct {
    LineWidth     int    // chars per line, default 80
    ScrollMode    string // "line" | "page"
    Theme         string // "dark" | "light" | "solarized" | custom
    VimKeys       bool
    AutoSave      bool
    AutoSaveEvery int    // paragraphs between auto-saves
}
```

---

## Database Schema (written, pending approval)

Files created: [`internal/storage/migrations/001_initial.sql`](./internal/storage/migrations/001_initial.sql), [`002_indexes.sql`](./internal/storage/migrations/002_indexes.sql)

### Tables
| Table | PK | Notes |
|---|---|---|
| `schema_version` | — | Single row, monotonic int |
| `sources` | `id TEXT` | Synced from `sources/` dir on startup |
| `novels` | `id TEXT` UUID | `UNIQUE(source_id, source_url)`; `in_library` bool allows browse-without-add |
| `novel_tags` | `(novel_id, tag)` | Normalized junction table |
| `chapters` | `id TEXT` UUID | `number REAL`; stub row exists before content fetched; `is_cached` flag |
| `reading_progress` | `novel_id TEXT` | One row per novel; upserted on save |
| `bookmarks` | `id TEXT` UUID | Indexed by `novel_id` and `chapter_id` |
| `history` | `id TEXT` UUID | Append-only log; `closed_at` NULL while session open |
| `user_settings` | `id INT CHECK(id=1)` | Single-row enforced by constraint; seeded with defaults in migration |

### Key schema decisions
- Datetimes stored as `TEXT` (ISO-8601 UTC) — SQLite has no native datetime; ISO-8601 sorts correctly as text
- `status CHECK(...)` — enum enforced at DB level
- `chapters.number REAL` — handles "ch 1.5" without extra columns
- `user_settings.id CHECK(id=1)` — DB-level single-row guard; `INSERT OR IGNORE` seeds defaults
- `history` partial index `WHERE closed_at IS NOT NULL` — keeps "Last Read" list query fast
- `PRAGMA foreign_keys = ON` + `PRAGMA journal_mode = WAL` set in migration

---

## Open Questions (blocking or pending decision)

| # | Question | Impact | Status |
|---|---|---|---|
| 1 | **Binary / app name**: use `novel` or something else? | Affects `go.mod` module path, binary name, install path | ✅ `novel`; module is `github.com/dharuncs/novel` |
| 2 | **`novel_tags` vs comma-separated `tags TEXT`**: normalized table is cleaner for queries; comma-separated is simpler CRUD. | Normalized `novel_tags` retained for future library filtering. | ✅ Confirmed |
| 3 | **`sources` table in DB vs in-memory only**: persisting to DB enables future per-source state. | `sources` retained; startup will upsert loaded plugin metadata by source ID. | ✅ Confirmed |

---

## Build Sequence (§7 of plan.md)

| Step | What | Status |
|---|---|---|
| **1** | `storage/` — SQLite schema, migrations, all CRUD | ✅ Complete; six temp-file SQLite tests pass |
| **2** | `source/engine.go` + `source/loader.go` — goja + Go `fetch` bridge | ✅ Complete; engine leak test passes |
| **3** | `sources/novelfire.js` — first plugin and CLI harness | ✅ Implemented; live diagnostic reaches pages 1–14 |
| **4** | `core/` — domain types + library/progress logic wired to storage | ✅ Complete & tested (4 unit tests passing) |
| **5** | `tui/library/` — bare list view | ✅ Complete (root AppModel + bare Library view) |
| **6** | `tui/reader/` — viewport, paragraph rendering, scrolling | ✅ Complete & tested (auto-save + resize stability) |
| **7** | Wire "continue reading" — position restore from DB | ✅ Complete (anchored paragraph seek on load) |
| **8** | `tui/search/` — search a source, add to library | ✅ Complete & tested |
| **9** | `tui/chapterlist/` — popup list, jump to chapter | ✅ Complete & tested |
| 10 | `tui/settings/` + themes | ⬜ Not started |
| 11 | Phase 2 features | ⬜ Not started |

---

## Environment

| Item | Status |
|---|---|
| Go installation | ✅ `go1.26.5 linux/amd64` |
| Project directory | `/home/dharun/novel` |
| `plan.md` | ✅ Written and up to date |
| `internal/storage/migrations/` | ✅ Created with 001 + 002 |
| `go.mod` | ✅ Initialized as `github.com/dharuncs/novel`; core dependencies installed (no `chromedp`) |
| Git checkpoint | ✅ Root commit `af02d27`; later diagnostic logging changes are uncommitted |

---

## Key Go Dependencies (planned)

```
github.com/charmbracelet/bubbletea    # TUI framework
github.com/charmbracelet/lipgloss     # Styling / themes
github.com/charmbracelet/bubbles      # Pre-built components (list, viewport, textinput)
github.com/dop251/goja                # JS engine for source plugins
github.com/PuerkitoBio/goquery        # HTML parsing in Go fetch bridge
modernc.org/sqlite                    # Pure-Go SQLite (no cgo)
github.com/spf13/cobra                # CLI subcommands
github.com/BurntSushi/toml            # Config file parsing
github.com/chromedp/chromedp          # Headless browser fallback (Phase 3, opt-in)
github.com/sahilm/fuzzy               # Fuzzy filter in library/search views
```

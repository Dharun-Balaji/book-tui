# `novel` — TUI Novel Reader: Architecture & Implementation Plan

> **Stack**: Go + Bubble Tea · **Plugin Engine**: JS via `goja` · **Storage**: SQLite  
> **Target**: Linux primary, SSH-compatible, single binary + `sources/` directory

## Current Implementation Status

- Storage, migrations, CRUD, and six temp-file SQLite tests are complete.
- The goja engine, loader, registry, scraper client, and HTML selector bridge are complete.
- The engine uses a 30-second fetch-idle watchdog plus a five-minute outer ceiling; cleanup is covered by a goroutine test.
- `sources/novelfire.js` is the first implemented plugin; its CLI harness is `novel source test novelfire --url=...`.
- Live diagnostics reached Novelfire pages 1–14 (100 chapters per page, HTTP 200); the external command window ended before page 15.
- `internal/core/`, TUI packages, config, Royal Road, and bookmark CRUD remain unimplemented.

---

## 1. Language & Framework Validation

You chose **Go + Bubble Tea**. Here's why that's the right call over the alternatives:

| Dimension | Python + Textual | **Go + Bubble Tea** | Rust + Ratatui |
|---|---|---|---|
| TUI maturity | Excellent (Textual is the richest) | Very good (Bubbletea + Lip Gloss + Bubbles) | Good (Ratatui is solid) |
| Async / concurrency | asyncio, but GIL limits true parallelism | Goroutines — trivial, lightweight, idiomatic | tokio async, but steeper |
| HTML scraping | Beautiful Soup (best-in-class) | `goquery` (jQuery-like, excellent) | `scraper` crate (decent) |
| JS rendering fallback | `playwright-python` (easy) | `chromedp` or `rod` (well-supported) | `chromiumoxide` (usable) |
| Startup time | 200–800ms (Python import overhead) | **~5ms** (compiled binary) | ~5ms (compiled binary) |
| Binary distribution | Needs Python + venv, painful | **Single binary** (go build) | Single binary |
| Dev velocity (student) | Fastest to prototype | Fast, simple tooling, great stdlib | Slower (borrow checker learning curve) |
| Plugin scripting (goja/JS) | N/A | `goja` embeds V8-compatible JS natively | Difficult to embed safely |

**Verdict**: Go + Bubble Tea wins on the exact axes that matter here — instant startup, goroutine concurrency for background scraping, excellent `goquery` for HTML parsing, `goja` for the JS plugin engine (matching your LNReader-style plugin choice), and a single distributable binary. Rust would give marginally better performance you'll never notice in a TUI; Python would make the plugin scripting easier but startup and distribution worse.

---

## 2. Architecture Plan

### 2.1 Module / Package Breakdown

```
novel/
├── cmd/novel/          # main entrypoint — wires everything together
├── internal/
│   ├── tui/            # All Bubble Tea models, views, key handling
│   │   ├── app.go      # Root model, view routing
│   │   ├── library/    # Library view model
│   │   ├── reader/     # Reader view model (the core UX)
│   │   ├── search/     # Search/browse view model
│   │   ├── settings/   # Settings panel model
│   │   └── styles/     # Lip Gloss styles, theme definitions
│   ├── core/           # Pure domain logic, no UI or I/O
│   │   ├── novel.go    # Novel, Chapter, Bookmark domain types
│   │   ├── library.go  # Library management logic
│   │   ├── progress.go # ReadingProgress, history logic
│   │   └── stats.go    # Reading stats computation
│   ├── storage/        # Persistence layer (SQLite)
│   │   ├── db.go       # Connection, migrations, schema
│   │   ├── novels.go   # Novel CRUD
│   │   ├── chapters.go # Chapter CRUD + cache
│   │   ├── progress.go # ReadingProgress queries
│   │   └── settings.go # User settings storage
│   ├── source/         # Plugin engine + source interface
│   │   ├── plugin.go   # Plugin interface definition
│   │   ├── loader.go   # Discovers + loads .js files from sources/
│   │   ├── engine.go   # goja runtime wrapper, sandboxed JS execution
│   │   ├── registry.go # In-memory registry of loaded sources
│   │   └── schema.go   # Go types that JS plugins must produce
│   ├── scraper/        # HTTP + HTML layer
│   │   ├── client.go   # Shared HTTP client (rate limit, retry, headers)
│   │   ├── fetcher.go  # Plain HTTP fetch + goquery helper
│   │   ├── browser.go  # Optional headless fallback (chromedp/rod)
│   │   └── cache.go    # On-disk HTML cache (avoid refetch)
│   └── config/         # App configuration (TOML file)
│       ├── config.go
│       └── defaults.go
├── sources/            # Shipped JS plugin files (one per site)
│   └── novelfire.js    # first plugin; needsJS: false
├── go.mod
├── go.sum
└── README.md
```

---

### 2.2 Data Models

#### `Novel`
```go
type Novel struct {
    ID          string    // UUID, stable across renames
    SourceID    string    // Which plugin sourced this
    SourceURL   string    // Canonical URL on the source site
    Title       string
    Author      string
    Cover       string    // URL or local cache path
    Description string
    Tags        []string
    Status      string    // "ongoing" | "completed" | "hiatus"
    TotalChapters int
    AddedAt     time.Time
    UpdatedAt   time.Time // last metadata refresh
    InLibrary   bool
}
```

#### `Chapter`
```go
type Chapter struct {
    ID          string    // UUID
    NovelID     string    // FK → Novel
    SourceURL   string
    Number      float64   // float to handle "ch 1.5" extras
    Title       string
    Content     string    // Cleaned plain text (post-scrape)
    WordCount   int
    FetchedAt   time.Time
    IsCached    bool
}
```

#### `ReadingProgress`
```go
type ReadingProgress struct {
    NovelID      string
    ChapterID    string
    ChapterNum   float64
    // Position within chapter:
    ParagraphIdx int     // which paragraph (for "continue reading")
    ScrollOffset int     // pixel/line offset within that paragraph
    ProgressPct  float64 // 0.0–1.0 within chapter
    LastReadAt   time.Time
    // Overall novel progress:
    ChaptersRead int
    TotalReadSec int64   // cumulative reading seconds for stats
}
```

> **Why `ParagraphIdx` + `ScrollOffset` separately?**  
> Terminal reflow changes line counts when window is resized. Anchoring to
> paragraph index is resize-stable; the scroll offset within that paragraph
> is a best-effort fine position.

#### `Bookmark`
```go
type Bookmark struct {
    ID          string
    NovelID     string
    ChapterID   string
    ParagraphIdx int
    Note        string    // optional annotation
    CreatedAt   time.Time
}
```

#### `HistoryEntry`
```go
type HistoryEntry struct {
    NovelID    string
    ChapterID  string
    OpenedAt   time.Time
    ClosedAt   time.Time
    SessionSec int64
}
```

#### `Source` (plugin metadata, not the JS script itself)
```go
type Source struct {
    ID          string   // e.g. "royalroad"
    Name        string   // "Royal Road"
    Version     string   // semver from plugin header
    BaseURL     string
    Language    string
    NeedsJS     bool     // hint: use headless fallback?
    RateLimit   int      // requests/minute
    Loaded      bool     // runtime state
}
```

#### `UserSettings`
```go
type UserSettings struct {
    LineWidth    int      // chars per line, default 80
    ScrollMode   string   // "line" | "page"
    Theme        string   // "dark" | "light" | "solarized" | custom
    VimKeys      bool
    FontSize     int      // for future sixel/kitty support
    AutoSave     bool     // auto-save progress every N paragraphs
    AutoSaveEvery int
}
```

---

### 2.3 Storage: Why SQLite

| Option | Pros | Cons |
|---|---|---|
| **SQLite** | ACID, queryable, relational joins, future-proof for sync | One extra dep (`mattn/go-sqlite3` or `modernc/sqlite`) |
| JSON/TOML | Zero dep, human-readable | No efficient queries, entire file re-written per save, terrible for history logs |
| BoltDB/bbolt | Pure Go, embedded KV store | No SQL, harder to query across entities |

**Decision: SQLite via `modernc/sqlite`** (pure Go, no cgo, cross-platform, no gcc needed). The schema is relational by nature (novels → chapters → progress → history). SQLite gives us free indexing, migrations, and a clean path to future sync (export/import DB file, or replicate via Litestream).

**Migrations**: Use a simple in-code migration table (`schema_version`). On startup, apply any pending migrations in order. Library: `golang-migrate` or hand-rolled (hand-rolled is simpler here).

---

### 2.4 "Continue Reading" — Persistence & Restore Flow

```
On Close / Auto-Save:
  ReadingProgress.ParagraphIdx = current viewport top paragraph
  ReadingProgress.ScrollOffset = lines scrolled within that paragraph
  ReadingProgress.LastReadAt = now()
  → UPDATE reading_progress WHERE novel_id = ?

On Launch / Open Novel:
  1. Query ReadingProgress for novel
  2. Load Chapter by ChapterID
  3. Render chapter content → paragraph slice
  4. Seek viewport to ParagraphIdx
  5. Apply ScrollOffset
  → User sees exactly where they left off
```

**Auto-save trigger**: every 5 paragraphs scrolled (configurable) + on any navigation away from reader + on app exit (Bubble Tea's `tea.Quit` message → flush before exit).

---

### 2.5 Plugin System — JS-in-Go via `goja`

#### Why `goja` over `gopher-lua`?
- JS is far more familiar to web/scraping developers (most scraping snippets online are JS)
- `goja` is a full ES5.1 + partial ES6 implementation — no npm, no node, just a sandboxed runtime
- LNReader's actual plugins are JS — you can port/adapt them directly
- `gopher-lua` would require learning Lua for plugin authors

#### Plugin Spec (JS contract)

Every `.js` file in `sources/` must export a top-level object conforming to:

```javascript
// sources/royalroad.js
const source = {
  // Metadata (read by loader from header comment or exported object)
  id: "royalroad",
  name: "Royal Road",
  version: "1.0.0",
  baseURL: "https://www.royalroad.com",
  language: "en",
  rateLimit: 60,       // requests per minute
  needsJS: false,      // set true to trigger headless fallback

  // --- Required Methods ---

  // Returns: Novel[]
  search(query, page) { ... },

  // Returns: Novel (full metadata + chapter list stub)
  novelInfo(novelURL) { ... },

  // Returns: Chapter[] (stubs — url, number, title, no content)
  chapterList(novelURL) { ... },

  // Returns: string (cleaned plain text content)
  chapterContent(chapterURL) { ... },
};
```

**Go bridge**: The `engine.go` wrapper exposes a `fetch(url, options)` function into the JS sandbox. This function calls back into Go's HTTP client (with rate-limiting, retries, caching). The JS plugin never makes raw HTTP calls — all network goes through the Go bridge. This gives us:
- Centralized rate limiting per source
- Retry + exponential backoff in Go
- HTML caching at the Go layer
- Easy mocking for tests

```
JS Plugin
  └── calls fetch(url) → Go bridge (engine.go)
        └── scraper/client.go (rate limiter, retry)
              └── scraper/fetcher.go or scraper/browser.go
                    └── returns HTML string back to JS
JS Plugin
  └── parses HTML with built-in DOMParser or regex
  └── returns structured object → Go unmarshals via goja
```

#### Plugin Loading Flow
```
startup
  └── source/loader.go scans sources/ directory
        └── for each *.js file:
              ├── read header metadata (id, version, etc.)
              ├── create isolated goja runtime
              ├── inject Go bridge functions (fetch, log)
              ├── execute script → extract source object
              └── register in source/registry.go
```

#### Headless Fallback (for JS-heavy sites)
- `source.needsJS = true` → fetcher routes through `scraper/browser.go`
- Uses `chromedp` (headless Chrome via CDP) — launched lazily only when needed
- Browser instance is reused across requests to the same source
- Adds ~200MB memory only when active — acceptable since it's opt-in

---

## 3. Feature Roadmap

### MVP — v0.1 (Build This First)

> Goal: a working reader you'd actually use daily.

- [ ] **Storage layer** — SQLite schema, migrations, all CRUD
- [x] **Plugin engine** — `goja` loader, Go bridge (`fetch`), one working source (Novelfire)
- [ ] **Library view** — list of tracked novels, basic add/remove
- [ ] **Reader view** — paginated/scrollable chapter text, line width config
- [ ] **Continue Reading** — paragraph-anchored position save/restore
- [ ] **Chapter navigation** — prev/next, chapter list popup
- [ ] **Search view** — search a source, add result to library
- [ ] **Keyboard navigation** — arrow keys, j/k, g/G, q to quit
- [ ] **Basic themes** — dark / light via Lip Gloss
- [ ] **Single binary** — `go build`, sources/ directory beside it

### Phase 2 — v0.2 (Polish & Power Features)

- [ ] Fuzzy search/filter in library view (fzf-style, `go-fuzzy` or manual)
- [ ] Bookmarks — mark paragraph, list/jump to bookmarks
- [ ] Reading progress % in library view
- [ ] Chapter download (cache all chapters of a novel for offline)
- [ ] New chapter check (compare fetched chapter count vs stored)
- [ ] Vim keybinding mode toggle
- [ ] Reading stats view (words read, time spent, session history)
- [ ] Second source plugin (NovelBin or LightNovelPub — plain HTTP)
- [ ] **`novelfire.js` plugin** — plain HTTP path, realistic browser headers, paginated chapter list
- [ ] Settings TUI panel (line width, theme, scroll mode, vim keys)

### Phase 3 — v0.3 (Advanced)

- [ ] Headless JS fallback (`chromedp`) for WebNovel-type sites
- [ ] Cover image display (Kitty protocol / sixel for capable terminals)
- [ ] Background chapter pre-fetch (next N chapters silently)
- [ ] Plugin update mechanism (check version header vs installed)
- [ ] Export reading list (JSON/CSV for backup)
- [ ] Config file (`~/.config/novel/config.toml`) for persistent settings
- [ ] macOS / Windows testing + README notes

### Phase 4 — Future / Nice-to-Have

- [ ] Sync via file export (designed-for hook in storage layer)
- [ ] Annotations on bookmarks (mini notepad per chapter)
- [ ] Reading goals / streaks
- [ ] Community plugin repository (a `sources/` GitHub repo, `novel update-sources`)
- [ ] OPDS catalog support (for ebook servers)

---

## 4. Key Technical Risks & Decisions

### Risk 1: Anti-Scraping Measures
**Problem**: Cloudflare, CAPTCHAs, rate-limit 429s, bot-detection via missing/wrong headers.  
**Mitigations**:
- Default: realistic browser `User-Agent`, `Accept`, `Accept-Language`, `Sec-Fetch-*` headers on every request — the shared `client.go` sets these globally
- Per-source `rateLimit` field respected strictly (token bucket in `client.go`)
- Genuinely JS-challenge-gated sites → headless fallback (set `needsJS: true`)
- Exponential backoff on 429/503: 1s → 2s → 4s → 8s, then surface error to TUI
- Expose a `retryPolicy` config per source in the JS plugin header

> **novelfire.net specifics**: Bare requests (no `User-Agent` etc.) get a Cloudflare “Attention Required” block page (HTTP 403). A plain HTTP request with realistic browser headers (`User-Agent: Chrome/126`, `Accept`, `Referer`, `Sec-Fetch-*`) returns **HTTP 200 with fully server-rendered HTML** — chapter links are present directly in the markup, no JS execution required. The page does include a Cloudflare beacon `<script>` tag (passive telemetry, not a JS challenge). `novelfire.js` therefore sets `needsJS: false` and `rateLimit: 30`. No `chromedp` needed; the shared HTTP client’s header defaults handle it.

### Risk 2: HTML Structure Changes Breaking Plugins
**Problem**: Sites redesign → selectors break → plugin returns empty/garbage.  
**Mitigations**:
- Plugin system is explicitly community-maintained (like LNReader)
- Validate plugin output schema in Go (`schema.go`) — if required fields are missing, surface a clear "source broken" error, not a crash
- Add a `test` subcommand: `novel source test royalroad` runs a fixed URL through the plugin and validates output — makes regression-testing plugins trivial
- Version the plugin spec: if a breaking change is needed, bump `pluginSpecVersion` so old plugins fail cleanly with a useful message

### Risk 3: `goja` JS Compatibility
**Problem**: Plugin authors might use modern ES6+ features `goja` doesn't support.  
**Mitigations**:
- Document the supported JS subset (ES5.1 + ES6 class, arrow functions, template literals, Promises via polyfill)
- `goja` supports most practical ES6 features — array destructuring, `const/let`, spread — which covers all real scraping use cases
- No `fetch` in browser-API sense — all HTTP goes through the Go bridge, which sidesteps the problem entirely

### Risk 4: Terminal Rendering Edge Cases
**Problem**: Line-wrapping, ANSI codes in scraped content, Unicode CJK (wide chars), small terminals.  
**Mitigations**:
- Use `go-runewidth` (already used by Bubble Tea internally) for CJK-aware width calculation
- Strip all HTML tags + ANSI codes from chapter content in Go before storing — `chapter.Content` is always clean plain text + minimal markdown (`*italic*`, `**bold**`)
- Minimum supported terminal: 60 cols wide — display a warning below that
- Dynamically reflow text on terminal resize (Bubble Tea sends `tea.WindowSizeMsg`)

### Risk 5: goja Runtime Isolation
**Problem**: A malicious plugin could theoretically do damage via the JS sandbox.  
**Mitigations**:
- JS sandbox has NO access to filesystem, no `require()`, no `exec` — only the `fetch()` bridge Go exposes
- `fetch()` bridge enforces a per-source domain allowlist (the `baseURL` from metadata) — a plugin can only fetch from its own site
- Set a goja execution timeout (e.g., 30s) so a runaway plugin can't hang the TUI

### Risk 6: Resize-Stable Position Tracking
**Problem**: User resizes terminal → line count changes → scroll position jumps.  
**Mitigation**: As noted in the data model, position is stored as **paragraph index** not line number. On resize, the reader re-renders from scratch and seeks to the stored paragraph. This is exactly how browser-based readers handle reflow.

### Decision: `modernc/sqlite` vs `mattn/go-sqlite3`
- `mattn/go-sqlite3` requires `cgo` → needs `gcc`, complicates cross-compilation
- `modernc/sqlite` is pure Go, same SQL API, slightly larger binary (~3MB) — **choose this**
- Enables `GOARCH=arm64 GOOS=darwin go build` without a cross-compiler

---

## 5. Project Structure (Starting Point)

```
novel/
├── cmd/
│   └── novel/
│       └── main.go               # cobra root command, wires DI, starts Bubble Tea
│
├── internal/
│   ├── tui/
│   │   ├── app.go                # Root tea.Model — owns view stack, routes messages
│   │   ├── library/
│   │   │   ├── model.go          # Library view state
│   │   │   └── view.go           # Render function + keybindings
│   │   ├── reader/
│   │   │   ├── model.go          # Reader state (viewport, chapter, position)
│   │   │   ├── view.go           # Render chapter text with wrapping
│   │   │   └── keybinds.go       # j/k/g/G/b/: etc.
│   │   ├── search/
│   │   │   ├── model.go
│   │   │   └── view.go
│   │   ├── chapterlist/
│   │   │   ├── model.go
│   │   │   └── view.go
│   │   ├── settings/
│   │   │   ├── model.go
│   │   │   └── view.go
│   │   └── styles/
│   │       ├── themes.go         # Lip Gloss color palettes
│   │       └── common.go         # Shared style vars
│   │
│   ├── core/
│   │   ├── novel.go              # Domain types (Novel, Chapter, Bookmark...)
│   │   ├── library.go            # Business logic: add/remove/update novel
│   │   ├── progress.go           # Save/restore ReadingProgress
│   │   └── stats.go              # Reading stats aggregation
│   │
│   ├── storage/
│   │   ├── db.go                 # Open DB, run migrations
│   │   ├── migrations/
│   │   │   ├── 001_initial.sql
│   │   │   └── 002_indexes.sql
│   │   ├── novels.go             # Novel CRUD
│   │   ├── chapters.go           # Chapter CRUD + cache check
│   │   ├── progress.go           # ReadingProgress queries
│   │   ├── history.go            # History log
│   │   └── settings.go           # User settings R/W
│   │
│   ├── source/
│   │   ├── plugin.go             # Source interface + metadata types
│   │   ├── loader.go             # Scan sources/ dir, load .js files
│   │   ├── engine.go             # goja runtime, Go bridge injection, timeout
│   │   ├── registry.go           # Map[sourceID]→LoadedPlugin
│   │   └── schema.go             # Go structs that JS results are decoded into
│   │
│   ├── scraper/
│   │   ├── client.go             # http.Client + rate limiter (token bucket)
│   │   ├── fetcher.go            # Plain HTTP fetch, returns HTML string
│   │   ├── browser.go            # chromedp headless fallback
│   │   └── cache.go              # Hash-keyed on-disk HTML cache
│   │
│   └── config/
│       ├── config.go             # TOML config struct
│       └── defaults.go           # Sensible defaults
│
├── sources/                      # JS plugin files — shipped with binary
│   └── novelfire.js              # needsJS: false (plain HTTP + browser headers)
│
├── go.mod
├── go.sum
├── Makefile                      # build, test, lint targets
└── README.md
```

### Key Go Dependencies (go.mod)

```
github.com/charmbracelet/bubbletea        # TUI framework
github.com/charmbracelet/lipgloss         # Styling
github.com/charmbracelet/bubbles          # Pre-built components (list, viewport, textinput)
github.com/dop251/goja                    # JS engine for plugins
github.com/PuerkitoBio/goquery            # jQuery-like HTML parsing (used in Go bridge)
modernc.org/sqlite                        # Pure-Go SQLite driver
github.com/spf13/cobra                    # CLI command routing
github.com/BurntSushi/toml                # Config file parsing
# chromedp is intentionally deferred until Phase 3
github.com/sahilm/fuzzy                   # Fuzzy search in library
```

---

## 6. Bubble Tea Architecture Notes

Bubble Tea uses the **Elm Architecture** (Model → View → Update). Here's how the app's view stack works:

```
AppModel (root)
  ├── state: enum { Library, Reader, Search, ChapterList, Settings }
  └── sub-models:
        ├── LibraryModel      ← active when state = Library
        ├── ReaderModel       ← active when state = Reader
        ├── SearchModel       ← active when state = Search
        └── ...

Messages (tea.Msg) flow up from sub-models to AppModel:
  OpenNovelMsg{novelID}     → AppModel transitions to Reader, loads progress
  BackToLibraryMsg{}        → AppModel saves progress, transitions to Library
  FetchCompleteMsg{chapter} → ReaderModel receives chapter content
  ProgressSaveMsg{}         → triggers DB write (auto-save)

Background work (goroutines → tea.Cmd):
  All network/DB calls run as tea.Cmd (non-blocking)
  Results come back as tea.Msg — UI never blocks
```

---

## 7. First Build Sequence (Suggested Order)

> Build in this order to get a working app as fast as possible.

1. `storage/` — DB schema + migrations + all CRUD ✅
2. `source/engine.go` + `source/loader.go` — goja runtime + Go `fetch` bridge ✅
3. `sources/novelfire.js` — first plugin, test with `novel source test novelfire` ✅
4. `core/` — domain types + library/progress logic wired to storage ✅
5. `tui/library/` — bare list view, just shows novels in DB ✅
6. `tui/reader/` — viewport, paragraph rendering, j/k/g/G scrolling ✅
7. Wire "continue reading" — open novel → restore position from DB ✅
8. `tui/search/` — search a source, add to library ✅
9. `tui/chapterlist/` — popup list, jump to chapter ✅
10. `tui/settings/` + themes — dark/light themes, line width, auto-save interval ✅
11. Phase 2 features...

# VisiData Go Port — Implementation Plan

> A comprehensive plan for reimplementing VisiData as a single Go binary that carries out all/most of the current functionality.

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Current Architecture Analysis](#2-current-architecture-analysis)
3. [Go Port Architecture](#3-go-port-architecture)
4. [Module Mapping: Python → Go](#4-module-mapping-python--go)
5. [Recommended Go Libraries](#5-recommended-go-libraries)
6. [Implementation Phases](#6-implementation-phases)
7. [Detailed Component Design](#7-detailed-component-design)
8. [File Format / Loader Coverage](#8-file-format--loader-coverage)
9. [Feature Parity Matrix](#9-feature-parity-matrix)
10. [Risks, Trade-offs, and Decisions](#10-risks-trade-offs-and-decisions)
11. [Build, Release, and Distribution](#11-build-release-and-distribution)
12. [Testing Strategy](#12-testing-strategy)
13. [Estimated Effort](#13-estimated-effort)

---

## 1. Executive Summary

**Goal**: Reimplement VisiData in Go so that a **single statically-linked binary** can provide interactive, terminal-based data exploration for CSV, JSON, SQLite, Excel, and many other formats — without requiring Python or any runtime dependencies.

**Why Go?**
- Single binary distribution (no Python/pip/virtualenv)
- Fast startup (~10ms vs ~500ms for Python)
- Built-in concurrency via goroutines (replaces Python's threading)
- Cross-compilation for Linux/macOS/Windows/ARM
- Strong standard library for CSV, JSON, SQL, HTTP, compression

**Scope**: The plan covers a phased approach — from a working MVP with core formats (CSV, JSON, TSV, SQLite) to full feature parity with the Python version's ~70 loaders, ~60 features, and 3 standalone apps.

**Current VisiData size**: ~33,000 lines of Python (18k core + 10k loaders + 5k features). Estimated Go equivalent: ~40,000–50,000 lines (Go is typically more verbose but explicit).

---

## 2. Current Architecture Analysis

### 2.1 Repository Structure (Python)

```
visidata/
├── visidata/           # 18,237 lines — core package
│   ├── basesheet.py    # BaseSheet, DrawablePane base classes
│   ├── sheets.py       # TableSheet — main tabular sheet (1,398 lines)
│   ├── column.py       # Column types, caching, formatting (628 lines)
│   ├── settings.py     # Options, commands, settings manager (614 lines)
│   ├── mainloop.py     # Curses event loop, screen drawing
│   ├── _input.py       # Keyboard/mouse input, modal editor (704 lines)
│   ├── threads.py      # @asyncthread, Progress, thread pool (546 lines)
│   ├── cmdlog.py       # Command log, replay, undo (494 lines)
│   ├── path.py         # File path abstraction (541 lines)
│   ├── cliptext.py     # Terminal text rendering with color (534 lines)
│   ├── menu.py         # Menu system (517 lines)
│   ├── expr.py         # Expression column evaluation (128 lines)
│   ├── aggregators.py  # sum, avg, median, etc. (422 lines)
│   ├── selection.py    # Row selection (266 lines)
│   ├── sort.py         # Multi-column sort (177 lines)
│   ├── movement.py     # Cursor/viewport navigation (233 lines)
│   ├── undo.py         # Undo/redo system (125 lines)
│   ├── clipboard.py    # Internal + system clipboard (256 lines)
│   ├── _types.py       # Type registry (int, float, date, currency)
│   ├── color.py        # Color theme management
│   ├── canvas.py       # ASCII/graphical canvas (879 lines)
│   └── ...             # 30+ more core modules
│
├── visidata/loaders/   # 9,630 lines — 69 format loaders
│   ├── csv.py, tsv.py, json.py, sqlite.py      # Built-in formats
│   ├── xlsx.py, parquet.py, hdf5.py, yaml.py   # Library-dependent
│   ├── postgres.py, mysql.py                     # Database connectors
│   ├── http.py, s3.py                           # Remote sources
│   ├── api_reddit.py, api_zulip.py, ...         # API integrations
│   └── ...
│
├── visidata/features/  # 4,809 lines — 60 auto-loaded feature modules
│   ├── join.py         # Multi-sheet joins
│   ├── describe.py     # Statistical summary
│   ├── transpose.py    # Row/column transpose
│   ├── dedupe.py       # Deduplication
│   ├── freeze.py       # Freeze rows/columns
│   ├── melt.py         # Unpivot/melt
│   └── ...
│
└── visidata/apps/      # 3 standalone applications
    ├── vgit/           # Terminal Git client
    ├── vdsql/          # SQL IDE (Ibis-based)
    └── galcon/         # Galactic Conquest game
```

### 2.2 Key Architectural Patterns

| Pattern | Python Approach | Go Equivalent |
|---------|----------------|---------------|
| **Extensibility** | `Extensible` base class, `@api` decorators, monkey-patching | Interfaces, plugin registry, functional options |
| **Concurrency** | `@asyncthread` decorator → daemon threads | Goroutines + channels + context.Context |
| **Type system** | Pluggable `typemap` dict: `type → VisiDataType` | Interface-based type registry with `TypeConverter` |
| **Command system** | `addCommand(key, name, execstr)` with `eval()` | Command registry: `map[string]CommandFunc` |
| **Options** | `SettingsMgr` with hierarchical context lookup | Struct-based options with merge semantics |
| **Expression eval** | Python `compile()` + `eval()` with lazy row context | Embedded expression engine (e.g., `expr` or `govaluate`) |
| **Row identity** | `id(row)` — Python object identity | Explicit row ID (auto-increment or pointer address) |
| **Sheet stack** | Python list as stack, `vd.push(sheet)` | Slice-based stack with `Push()/Pop()` |
| **Progress tracking** | `Progress` context manager on thread | Channel-based progress reporting |
| **Undo** | Closure stack per command | Command pattern with `Undo()` interface |

### 2.3 External Dependencies Summary

**Required (core)**: `python-dateutil` only.
**Optional (loaders)**: 51 packages for format/API support.
**Key insight**: VisiData's graceful degradation means most loaders fail with a helpful message if the package is missing. Go can compile in all format support statically.

---

## 3. Go Port Architecture

### 3.1 Proposed Directory Layout

```
vdgo/
├── cmd/
│   └── vd/
│       └── main.go              # Entry point, CLI flags, initialization
│
├── internal/
│   ├── core/
│   │   ├── app.go               # VisiData singleton (vd object equivalent)
│   │   ├── sheet.go             # BaseSheet, Sheet interfaces
│   │   ├── tablesheet.go        # TableSheet with rows/columns
│   │   ├── column.go            # Column interface + implementations
│   │   ├── command.go           # Command registry + execution
│   │   ├── options.go           # Hierarchical options system
│   │   ├── types.go             # Type registry (int, float, date, etc.)
│   │   ├── row.go               # Row abstraction with stable ID
│   │   ├── selection.go         # Row selection (map-based)
│   │   ├── sort.go              # Multi-column sorting
│   │   ├── aggregator.go        # Aggregation functions
│   │   ├── undo.go              # Undo/redo stack
│   │   ├── clipboard.go         # Internal + system clipboard
│   │   ├── path.go              # Path abstraction for data sources
│   │   └── expr.go              # Expression evaluation engine
│   │
│   ├── tui/
│   │   ├── app.go               # Main event loop (tcell-based)
│   │   ├── screen.go            # Screen management, viewports
│   │   ├── draw.go              # Sheet drawing (rows, columns, status)
│   │   ├── input.go             # Keyboard/mouse input handling
│   │   ├── color.go             # Color/theme management
│   │   ├── menu.go              # Menu system
│   │   ├── statusbar.go         # Status bar rendering
│   │   ├── sidebar.go           # Sidebar navigation
│   │   └── canvas.go            # Graphical canvas (ASCII plots)
│   │
│   ├── loader/
│   │   ├── registry.go          # Loader plugin registry
│   │   ├── csv.go               # CSV/TSV/PSV loader + saver
│   │   ├── json.go              # JSON/JSONL loader + saver
│   │   ├── sqlite.go            # SQLite loader + saver
│   │   ├── xlsx.go              # Excel XLSX loader
│   │   ├── parquet.go           # Parquet loader (via parquet-go)
│   │   ├── yaml.go              # YAML loader
│   │   ├── xml.go               # XML loader
│   │   ├── html.go              # HTML table loader
│   │   ├── fixed_width.go       # Fixed-width format
│   │   ├── postgres.go          # PostgreSQL connector
│   │   ├── mysql.go             # MySQL connector
│   │   ├── http.go              # HTTP/URL loader
│   │   ├── arrow.go             # Apache Arrow/IPC
│   │   ├── pdf.go               # PDF table extraction
│   │   ├── archive.go           # ZIP/TAR/GZ archives
│   │   ├── dir.go               # Directory listing
│   │   └── ...                  # Additional format loaders
│   │
│   ├── feature/
│   │   ├── join.go              # Multi-sheet joins
│   │   ├── describe.go          # Statistical summary
│   │   ├── pivot.go             # Pivot tables
│   │   ├── frequency.go         # Frequency tables
│   │   ├── transpose.go         # Transpose
│   │   ├── melt.go              # Unpivot/melt
│   │   ├── dedupe.go            # Deduplication
│   │   ├── freeze.go            # Freeze rows/columns
│   │   ├── regex.go             # Regex operations
│   │   ├── graph.go             # ASCII graphing
│   │   └── ...
│   │
│   └── metasheet/
│       ├── columns.go           # Columns sheet (inspect/edit columns)
│       ├── options.go           # Options sheet
│       ├── commands.go          # Commands sheet
│       ├── sheets.go            # Sheets sheet (all open sheets)
│       └── errors.go            # Error sheet
│
├── pkg/                         # Public API (if needed for plugins)
│   └── vdapi/
│       ├── interfaces.go        # Public Sheet/Column/Command interfaces
│       └── types.go             # Public type definitions
│
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

### 3.2 Core Interfaces

```go
// Sheet is the fundamental data container
type Sheet interface {
    Name() string
    Rows() []Row
    Columns() []Column
    Draw(screen Screen)
    Reload(ctx context.Context) error
    // Navigation
    CursorRow() int
    CursorCol() int
    MoveCursor(dRow, dCol int)
    // Selection
    Select(rows ...Row)
    Unselect(rows ...Row)
    SelectedRows() []Row
    // Commands
    ExecCommand(cmd string, args ...interface{}) error
}

// Column defines a data column with type, getter, formatter
type Column interface {
    Name() string
    Type() DataType
    Width() int
    GetValue(row Row) (interface{}, error)
    GetTypedValue(row Row) (interface{}, error)
    GetDisplayValue(row Row) string
    SetValue(row Row, val interface{}) error
}

// Row wraps a data row with a stable identity
type Row interface {
    ID() int64            // Stable identity (survives sort)
    Data() interface{}    // Underlying data ([]string, map, struct, etc.)
}

// Command represents a user-executable action
type Command struct {
    Key      string       // Keyboard shortcut
    LongName string       // Canonical name (e.g., "delete-row")
    Help     string       // Help text
    Exec     CommandFunc  // Function to execute
    Undo     UndoFunc     // Optional undo function
}
type CommandFunc func(ctx *CommandContext) error

// Loader opens a data source and returns a Sheet
type Loader interface {
    CanOpen(path string) bool
    Open(path string) (Sheet, error)
}

// Saver exports a Sheet to a file
type Saver interface {
    CanSave(path string) bool
    Save(path string, sheets ...Sheet) error
}
```

### 3.3 Concurrency Model

```
Python @asyncthread        →  Go goroutine + context.Context
Python Progress(iterable)  →  Go channel-based progress reporting
Python vd.sync(threads)    →  Go sync.WaitGroup / errgroup
Python thread cancellation →  Go context.WithCancel()
```

**Pattern**:
```go
func (s *TableSheet) Reload(ctx context.Context) error {
    progress := NewProgress(ctx, "loading", s)
    defer progress.Done()

    go func() {
        for row := range dataSource.Rows(ctx) {
            s.AddRow(row)
            progress.Increment()
        }
    }()
    return nil
}
```

---

## 4. Module Mapping: Python → Go

### 4.1 Core Modules

| Python Module | Lines | Go Package | Notes |
|---------------|-------|-----------|-------|
| `basesheet.py` | 387 | `core/sheet.go` | `Sheet` interface + `BaseSheet` struct |
| `sheets.py` | 1,398 | `core/tablesheet.go` | `TableSheet` struct with column/row management |
| `column.py` | 628 | `core/column.go` | `Column` interface + `ItemColumn`, `ExprColumn`, etc. |
| `settings.py` | 614 | `core/options.go`, `core/command.go` | Split into options + command registry |
| `vdobj.py` | 199 | `core/app.go` | `App` struct (global singleton) |
| `mainloop.py` | ~300 | `tui/app.go` | tcell event loop |
| `cliptext.py` | 534 | `tui/draw.go` | Text rendering with color attributes |
| `_input.py` | 704 | `tui/input.go` | Keyboard/mouse input, line editor |
| `threads.py` | 546 | `core/async.go` | Goroutine pool with progress tracking |
| `cmdlog.py` | 494 | `core/cmdlog.go` | Command log for replay/undo |
| `path.py` | 541 | `core/path.go` | Data source path abstraction |
| `color.py` | ~250 | `tui/color.go` | Color themes and attribute mapping |
| `menu.py` | 517 | `tui/menu.go` | Menu system |
| `expr.go` | 128 | `core/expr.go` | Expression engine (govaluate or custom) |
| `aggregators.py` | 422 | `core/aggregator.go` | Aggregation function registry |
| `selection.py` | 266 | `core/selection.go` | Row selection with map-based membership |
| `sort.py` | 177 | `core/sort.go` | Multi-column sort with `sort.Slice` |
| `movement.py` | 233 | `tui/movement.go` | Cursor/viewport navigation |
| `undo.py` | 125 | `core/undo.go` | Command-based undo/redo |
| `clipboard.py` | 256 | `core/clipboard.go` | Internal memory + `exec.Command` for system clipboard |
| `_types.py` | ~150 | `core/types.go` | Type registry + converters |
| `canvas.py` | 879 | `tui/canvas.go` | ASCII/braille canvas plotting |
| `main.py` | 562 | `cmd/vd/main.go` | CLI parsing with `cobra` or `pflag` |

### 4.2 Feature Modules

| Feature | Python File | Go File | Complexity |
|---------|------------|---------|------------|
| Join | `features/join.py` | `feature/join.go` | Medium — inner/outer/full/cross joins |
| Describe | `features/describe.py` | `feature/describe.go` | Low — statistical summary |
| Transpose | `features/transpose.py` | `feature/transpose.go` | Low — swap rows/columns |
| Melt | `features/melt.py` | `feature/melt.go` | Medium — unpivot operation |
| Dedupe | `features/dedupe.py` | `feature/dedupe.go` | Low — unique/duplicate detection |
| Freeze | `features/freeze.py` | `feature/freeze.go` | Low — snapshot rows/columns |
| Pivot | `pivot.py` (core) | `feature/pivot.go` | Medium — pivot table generation |
| FreqTable | `freqtbl.py` (core) | `feature/frequency.go` | Medium — frequency/histogram |
| Regex | `features/regex.py` | `feature/regex.go` | Low — regex split/capture |
| Graph | `graph.py` (core) | `feature/graph.go` | High — ASCII graphing with canvas |
| Command palette | `features/cmdpalette.py` | `tui/cmdpalette.go` | Medium — fuzzy search |

### 4.3 Loader Modules

| Format | Python Loader | Go Library | Priority |
|--------|--------------|------------|----------|
| CSV/TSV/PSV | `csv.py`, `tsv.py` | `encoding/csv` (stdlib) | P0 |
| JSON/JSONL | `json.py`, `jsonla.py` | `encoding/json` (stdlib) | P0 |
| SQLite | `sqlite.py` | `modernc.org/sqlite` (pure Go) or `mattn/go-sqlite3` (CGo) | P0 |
| Fixed-width | `fixed_width.py` | Custom implementation | P0 |
| XLSX | `xlsx.py` | `excelize` (`github.com/xuri/excelize/v2`) | P1 |
| YAML | `yaml.py` | `gopkg.in/yaml.v3` | P1 |
| XML | `xml.py` | `encoding/xml` (stdlib) | P1 |
| HTML tables | `html.py` | `golang.org/x/net/html` | P1 |
| Parquet | `parquet.go` | `github.com/parquet-go/parquet-go` | P1 |
| Arrow/IPC | `arrow.py` | `github.com/apache/arrow/go` | P2 |
| PostgreSQL | `postgres.py` | `github.com/lib/pq` or `pgx` | P2 |
| MySQL | `mysql.py` | `github.com/go-sql-driver/mysql` | P2 |
| HTTP/URL | `http.py` | `net/http` (stdlib) | P1 |
| TOML | `toml.py` | `github.com/BurntSushi/toml` | P2 |
| PDF tables | `pdf.py` | `github.com/ledongthuc/pdf` or `pdfcpu` | P3 |
| HDF5 | `hdf5.py` | `gonum.org/v1/hdf5` | P3 |
| GeoJSON | `geojson.py` | `encoding/json` (stdlib) | P2 |
| Shapefile | `shp.py` | `github.com/jonas-p/go-shp` | P3 |
| MsgPack | `msgpack.py` | `github.com/vmihailenco/msgpack/v5` | P3 |
| Archive (ZIP/TAR) | `archive.py` | `archive/zip`, `archive/tar` (stdlib) | P1 |
| Directory listing | `_dir.py` | `os`, `path/filepath` (stdlib) | P0 |
| Markdown | `markdown.py` | `github.com/yuin/goldmark` | P2 |
| VCF (vCard) | `vcf.py` | Custom parser | P3 |
| Graphviz | `graphviz.py` | Custom DOT parser | P3 |

---

## 5. Recommended Go Libraries

### 5.1 Terminal UI

| Library | Purpose | Why |
|---------|---------|-----|
| [`tcell/v2`](https://github.com/gdamore/tcell) | Terminal screen, keyboard, mouse | Most mature Go TUI library; cross-platform; supports wide chars, mouse, 256 colors, true color |
| [`tview`](https://github.com/rivo/tview) | Higher-level TUI widgets (optional) | Built on tcell; provides Table, Form, etc. — useful for rapid prototyping but may be too opinionated for VisiData's custom rendering |
| **Recommendation**: Use `tcell/v2` directly for maximum control, similar to how VisiData uses raw curses |

### 5.2 Data Formats

| Library | Format | Notes |
|---------|--------|-------|
| `encoding/csv` | CSV/TSV | Stdlib; configure delimiter |
| `encoding/json` | JSON/JSONL | Stdlib; stream with `json.Decoder` |
| `encoding/xml` | XML | Stdlib |
| `gopkg.in/yaml.v3` | YAML | Standard YAML library |
| `github.com/xuri/excelize/v2` | XLSX | Full Excel read/write, pure Go |
| `github.com/parquet-go/parquet-go` | Parquet | Pure Go Parquet reader/writer |
| `github.com/apache/arrow/go/v17` | Arrow/IPC | Official Apache Arrow for Go |
| `modernc.org/sqlite` | SQLite | Pure Go SQLite (no CGo needed for single binary) |
| `github.com/BurntSushi/toml` | TOML | Standard TOML library |
| `golang.org/x/net/html` | HTML parsing | For HTML table extraction |
| `github.com/ledongthuc/pdf` | PDF | PDF text extraction |

### 5.3 Database Connectors

| Library | Database | Notes |
|---------|----------|-------|
| `github.com/jackc/pgx/v5` | PostgreSQL | Pure Go, high performance |
| `github.com/go-sql-driver/mysql` | MySQL/MariaDB | Standard MySQL driver |
| `modernc.org/sqlite` | SQLite | Pure Go (preferred for single binary) |
| `database/sql` | Generic SQL | Stdlib interface for all databases |

### 5.4 Utilities

| Library | Purpose | Notes |
|---------|---------|-------|
| `github.com/spf13/cobra` | CLI framework | Industry standard for Go CLIs |
| `github.com/spf13/viper` | Configuration | Hierarchical config (matches VisiData's options) |
| `github.com/araddon/dateparse` | Date parsing | Auto-detect date formats |
| `github.com/dustin/go-humanize` | Number formatting | SI units, commas, etc. |
| `golang.org/x/text` | Unicode, locale | Wide char width, locale-aware formatting |
| `github.com/sahilm/fuzzy` | Fuzzy matching | For command palette |
| `github.com/expr-lang/expr` | Expression eval | Safe expression language (replaces Python eval) |
| `github.com/klauspost/compress` | Compression | zstd, gzip, snappy, etc. |

---

## 6. Implementation Phases

### Phase 0: Foundation (Weeks 1–3)

**Goal**: Minimal terminal app that opens and displays a CSV file with basic navigation.

- [ ] Project scaffolding (`go mod init`, directory structure, Makefile)
- [ ] `core/row.go` — Row abstraction with stable IDs
- [ ] `core/column.go` — Column interface + `ItemColumn` for `[]string` rows
- [ ] `core/tablesheet.go` — Basic `TableSheet` with rows/columns
- [ ] `core/types.go` — Type registry: `string`, `int`, `float`
- [ ] `tui/app.go` — tcell event loop, screen initialization
- [ ] `tui/draw.go` — Draw table: header row, data rows, column separators
- [ ] `tui/input.go` — Basic key handling (quit, scroll)
- [ ] `tui/movement.go` — Cursor movement (hjkl/arrows), page up/down
- [ ] `tui/statusbar.go` — Status bar (sheet name, row count, position)
- [ ] `loader/csv.go` — CSV loader (delimiter-configurable for TSV/PSV)
- [ ] `cmd/vd/main.go` — CLI entry point: `vdgo file.csv`

**Deliverable**: `vdgo sample.csv` opens a scrollable table view.

### Phase 1: Core Data Operations (Weeks 4–7)

**Goal**: Essential data exploration: types, sort, select, search, column operations.

- [ ] `core/options.go` — Options system with global/sheet-level hierarchy
- [ ] `core/command.go` — Command registry with keybindings
- [ ] `core/types.go` — Full type system: `date`, `currency`, `vlen`
- [ ] `core/sort.go` — Multi-column sort with `[` / `]` keys
- [ ] `core/selection.go` — Row selection (`s`/`t`/`u` keys, regex select)
- [ ] `core/aggregator.go` — Aggregators: `sum`, `avg`, `count`, `min`, `max`, `median`
- [ ] `tui/input.go` — Line editor for search/commands (input modal)
- [ ] `tui/color.go` — Color scheme + type-based coloring
- [ ] Column operations: hide (`-`), resize (`_`), rename (`^`), type change (`~#%$@`)
- [ ] Search: `/` (regex search), `n`/`N` (next/prev match)
- [ ] `loader/json.go` — JSON/JSONL loader
- [ ] `loader/fixed_width.go` — Fixed-width loader
- [ ] `loader/dir.go` — Directory listing

**Deliverable**: Full navigation, sorting, selection, type coercion, basic search.

### Phase 2: Advanced Features (Weeks 8–12)

**Goal**: Feature parity for data manipulation, metasheets, frequency tables, joins.

- [ ] `core/undo.go` — Undo/redo system
- [ ] `core/clipboard.go` — Copy/paste (internal + system clipboard)
- [ ] `core/cmdlog.go` — Command logging + replay
- [ ] `core/expr.go` — Expression column (`=expr` to add computed columns)
- [ ] `feature/frequency.go` — Frequency table (`F` key)
- [ ] `feature/describe.go` — Statistical describe sheet (`I` key)
- [ ] `feature/pivot.go` — Pivot tables (`W` key)
- [ ] `feature/join.go` — Sheet joins (`&` key)
- [ ] `feature/transpose.go` — Transpose (`T` key)
- [ ] `feature/melt.go` — Unpivot/melt (`M` key)
- [ ] `feature/dedupe.go` — Deduplication
- [ ] `metasheet/columns.go` — Columns sheet (`C` key)
- [ ] `metasheet/sheets.go` — Sheets sheet (`S` key)
- [ ] `metasheet/options.go` — Options sheet (`O` key)
- [ ] `metasheet/commands.go` — Commands/help sheet
- [ ] Save/export: `Ctrl+S` to save in current format, `save_tsv`, `save_csv`, `save_json`

**Deliverable**: Feature-complete data manipulation tool for text-based formats.

### Phase 3: Extended Loaders (Weeks 13–18)

**Goal**: Support for binary/structured formats and databases.

- [ ] `loader/xlsx.go` — Excel XLSX (read/write)
- [ ] `loader/sqlite.go` — SQLite (read/write, SQL queries)
- [ ] `loader/parquet.go` — Parquet (read)
- [ ] `loader/yaml.go` — YAML
- [ ] `loader/xml.go` — XML
- [ ] `loader/html.go` — HTML table extraction
- [ ] `loader/postgres.go` — PostgreSQL connector
- [ ] `loader/mysql.go` — MySQL connector
- [ ] `loader/http.go` — HTTP URL loader
- [ ] `loader/archive.go` — ZIP/TAR/GZ archives
- [ ] `loader/toml.go` — TOML
- [ ] `loader/markdown.go` — Markdown tables
- [ ] `loader/arrow.go` — Apache Arrow IPC
- [ ] Compression: transparent `.gz`, `.bz2`, `.xz`, `.zst` decompression

**Deliverable**: Handles 20+ file formats from a single binary.

### Phase 4: Advanced UI & Features (Weeks 19–24)

**Goal**: Full terminal UI with graphing, canvas, plugins, mouse support.

- [ ] `tui/canvas.go` — ASCII/braille canvas for graphing
- [ ] `feature/graph.go` — Scatter/line/bar charts
- [ ] `tui/menu.go` — Menu system with keyboard navigation
- [ ] `tui/cmdpalette.go` — Command palette with fuzzy search
- [ ] `tui/sidebar.go` — Sidebar for navigation
- [ ] Mouse support — Click, scroll, drag to select
- [ ] Multi-pane layout — Split screen with multiple sheets
- [ ] `feature/regex.go` — Regex capture/split columns
- [ ] `feature/freeze.go` — Freeze panes
- [ ] `feature/sparkline.go` — Sparkline columns
- [ ] Theme system — Load/switch color themes
- [ ] Configuration — `~/.vdgorc` configuration file
- [ ] Macros — Record and replay command sequences

**Deliverable**: Full-featured TUI data explorer.

### Phase 5: Apps & Ecosystem (Weeks 25–30)

**Goal**: Port standalone apps and ecosystem features.

- [ ] `apps/vgit/` — Git interface (using `os/exec` for git commands)
- [ ] `apps/vdsql/` — SQL query builder/IDE
- [ ] Plugin system — Load Go plugins or Lua/Wasm extensions
- [ ] API integrations — Reddit, S3, HTTP APIs
- [ ] Man page generation
- [ ] Shell completions (bash, zsh, fish)
- [ ] Homebrew formula, Snap, Docker, RPM/DEB packaging

**Deliverable**: Complete VisiData replacement as a single Go binary.

---

## 7. Detailed Component Design

### 7.1 Row Abstraction

Python VisiData uses `id(row)` (object identity) for stable row identification. Go needs an explicit approach:

```go
type Row struct {
    id   int64         // Auto-incremented, stable across sorts
    data interface{}   // Underlying data: []interface{}, map[string]interface{}, etc.
}

var nextRowID int64 // atomic counter

func NewRow(data interface{}) Row {
    return Row{
        id:   atomic.AddInt64(&nextRowID, 1),
        data: data,
    }
}
```

### 7.2 Column System

```go
type ColumnType int
const (
    TypeString ColumnType = iota
    TypeInt
    TypeFloat
    TypeDate
    TypeCurrency
    TypeLen
)

type BaseColumn struct {
    name       string
    colType    ColumnType
    width      int
    hidden     bool
    fmtstr     string
    getter     func(Row) (interface{}, error)
    setter     func(Row, interface{}) error
    aggregators []Aggregator
}

// ItemColumn gets values by index from []interface{} rows
type ItemColumn struct {
    BaseColumn
    index int
}

func (c *ItemColumn) GetValue(row Row) (interface{}, error) {
    data := row.data.([]interface{})
    if c.index >= len(data) {
        return nil, nil
    }
    return data[c.index], nil
}

// ExprColumn evaluates an expression per row
type ExprColumn struct {
    BaseColumn
    expr    string
    program *vm.Program // compiled expression (github.com/expr-lang/expr)
}
```

### 7.3 Options System

Mirrors VisiData's hierarchical resolution: default → global → sheet-type → instance.

```go
type OptionLevel int
const (
    OptDefault OptionLevel = iota
    OptGlobal
    OptSheetType
    OptInstance
)

type Option struct {
    Name        string
    Default     interface{}
    Description string
    Value       map[OptionLevel]interface{} // per-level overrides
}

type Options struct {
    mu      sync.RWMutex
    options map[string]*Option
}

func (o *Options) Get(name string, context ...OptionLevel) interface{} {
    // Walk from most specific to least specific
    for level := OptInstance; level >= OptDefault; level-- {
        if val, ok := o.options[name].Value[level]; ok {
            return val
        }
    }
    return o.options[name].Default
}
```

### 7.4 Command System

```go
type CommandContext struct {
    App       *App
    Sheet     Sheet
    CursorRow Row
    CursorCol Column
    Input     string // User input (if prompted)
}

type CommandRegistry struct {
    commands map[string]*Command      // longname → command
    bindings map[string]string        // key → longname
    prefixes map[string]string        // prefix combos (g, z, gz)
}

func (r *CommandRegistry) Register(cmd *Command) {
    r.commands[cmd.LongName] = cmd
    if cmd.Key != "" {
        r.bindings[cmd.Key] = cmd.LongName
    }
}

func (r *CommandRegistry) Exec(longname string, ctx *CommandContext) error {
    cmd, ok := r.commands[longname]
    if !ok {
        return fmt.Errorf("unknown command: %s", longname)
    }
    // Record in command log
    ctx.App.CmdLog.Record(longname, ctx)
    // Execute
    return cmd.Exec(ctx)
}
```

### 7.5 Async Operations

```go
type AsyncTask struct {
    ID        int64
    Name      string
    Sheet     Sheet
    Cancel    context.CancelFunc
    Progress  *Progress
    StartTime time.Time
    EndTime   time.Time
    Err       error
}

type Progress struct {
    total   int64
    current int64
    gerund  string
}

func (a *App) RunAsync(name string, sheet Sheet, fn func(ctx context.Context, p *Progress) error) *AsyncTask {
    ctx, cancel := context.WithCancel(context.Background())
    task := &AsyncTask{
        Name:      name,
        Sheet:     sheet,
        Cancel:    cancel,
        Progress:  &Progress{gerund: name},
        StartTime: time.Now(),
    }
    a.tasks = append(a.tasks, task)

    go func() {
        defer func() { task.EndTime = time.Now() }()
        task.Err = fn(ctx, task.Progress)
    }()

    return task
}
```

### 7.6 Expression Engine

Replace Python's `eval()` with a safe expression language:

```go
import "github.com/expr-lang/expr"

type ExprColumn struct {
    BaseColumn
    exprStr string
    program *vm.Program
}

func NewExprColumn(name, expression string, sheet *TableSheet) (*ExprColumn, error) {
    // Build environment from column names
    env := make(map[string]interface{})
    for _, col := range sheet.Columns() {
        env[col.Name()] = 0 // type hint
    }
    program, err := expr.Compile(expression, expr.Env(env))
    if err != nil {
        return nil, err
    }
    return &ExprColumn{
        BaseColumn: BaseColumn{name: name},
        exprStr:    expression,
        program:    program,
    }, nil
}

func (c *ExprColumn) GetValue(row Row) (interface{}, error) {
    env := make(map[string]interface{})
    for _, col := range c.sheet.Columns() {
        val, _ := col.GetValue(row)
        env[col.Name()] = val
    }
    return expr.Run(c.program, env)
}
```

### 7.7 Loader Registry

```go
type LoaderFunc func(path string) (Sheet, error)
type SaverFunc  func(path string, sheets ...Sheet) error

type LoaderRegistry struct {
    loaders map[string]LoaderFunc  // extension → loader
    savers  map[string]SaverFunc   // extension → saver
}

func (r *LoaderRegistry) Register(ext string, loader LoaderFunc, saver SaverFunc) {
    r.loaders[ext] = loader
    if saver != nil {
        r.savers[ext] = saver
    }
}

func (r *LoaderRegistry) Open(path string) (Sheet, error) {
    ext := filepath.Ext(path)
    // Check for compression (.csv.gz → decompress then open .csv)
    if isCompressed(ext) {
        reader, innerExt := decompress(path)
        ext = innerExt
    }
    loader, ok := r.loaders[ext]
    if !ok {
        return nil, fmt.Errorf("no loader for %s files", ext)
    }
    return loader(path)
}

// Registration (called from init() in each loader file)
func init() {
    DefaultRegistry.Register(".csv", OpenCSV, SaveCSV)
    DefaultRegistry.Register(".tsv", OpenTSV, SaveTSV)
    DefaultRegistry.Register(".json", OpenJSON, SaveJSON)
    // ...
}
```

---

## 8. File Format / Loader Coverage

### Priority Tiers

**P0 — MVP (must have for initial release)**:
| Format | Read | Write | Go Library | Effort |
|--------|------|-------|------------|--------|
| CSV | ✅ | ✅ | `encoding/csv` | Low |
| TSV | ✅ | ✅ | `encoding/csv` (tab delimiter) | Low |
| JSON | ✅ | ✅ | `encoding/json` | Low |
| JSONL | ✅ | ✅ | `encoding/json` (line-by-line) | Low |
| Fixed-width | ✅ | ✅ | Custom | Low |
| SQLite | ✅ | ✅ | `modernc.org/sqlite` | Medium |
| Directory | ✅ | — | `os`, `path/filepath` | Low |
| Stdin/pipe | ✅ | — | `os.Stdin` | Low |

**P1 — Essential (for broad adoption)**:
| Format | Read | Write | Go Library | Effort |
|--------|------|-------|------------|--------|
| XLSX | ✅ | ✅ | `excelize/v2` | Medium |
| YAML | ✅ | ✅ | `yaml.v3` | Low |
| XML | ✅ | — | `encoding/xml` | Medium |
| HTML tables | ✅ | — | `x/net/html` | Medium |
| Parquet | ✅ | ✅ | `parquet-go` | Medium |
| HTTP/URL | ✅ | — | `net/http` | Low |
| Archive (ZIP/TAR) | ✅ | — | `archive/*` | Low |
| Compressed (.gz, .bz2, .xz, .zst) | ✅ | ✅ | `compress/*`, `klauspost/compress` | Low |

**P2 — Extended (for power users)**:
| Format | Read | Write | Go Library | Effort |
|--------|------|-------|------------|--------|
| PostgreSQL | ✅ | — | `pgx/v5` | Medium |
| MySQL | ✅ | — | `go-sql-driver/mysql` | Medium |
| Arrow/IPC | ✅ | ✅ | `apache/arrow/go` | Medium |
| TOML | ✅ | — | `BurntSushi/toml` | Low |
| Markdown | ✅ | — | `goldmark` | Low |
| GeoJSON | ✅ | — | `encoding/json` | Low |
| S3 | ✅ | — | `aws-sdk-go-v2` | Medium |

**P3 — Niche (for completeness)**:
| Format | Go Approach |
|--------|------------|
| HDF5 | CGo wrapper or skip |
| PDF | `pdfcpu` or `ledongthuc/pdf` |
| Shapefile | `go-shp` |
| MsgPack | `vmihailenco/msgpack` |
| PCap | `google/gopacket` |
| VCard | Custom parser |
| PNG (pixel data) | `image/png` stdlib |
| TTF (font data) | `golang.org/x/image/font` |
| Org-mode | Custom parser |

---

## 9. Feature Parity Matrix

| Feature | Python VisiData | Go Port Plan | Phase |
|---------|----------------|-------------|-------|
| **Navigation** | hjkl, arrows, page, home/end | Same keybindings | 0 |
| **Column types** | string, int, float, date, currency, vlen | Same set | 1 |
| **Sort** | Multi-column, asc/desc | Same | 1 |
| **Selection** | Regex, expression, manual | Same | 1 |
| **Search** | Regex forward/backward | Same | 1 |
| **Column ops** | Hide, show, resize, rename, move | Same | 1 |
| **Aggregation** | sum, avg, count, min, max, median, stdev | Same | 1 |
| **Frequency table** | Group by column, show counts | Same | 2 |
| **Describe** | Statistical summary per column | Same | 2 |
| **Pivot** | Cross-tabulation | Same | 2 |
| **Join** | Inner, outer, full, cross, diff | Same | 2 |
| **Transpose** | Swap rows/columns | Same | 2 |
| **Melt/Unpivot** | Wide → long format | Same | 2 |
| **Dedupe** | Detect/remove duplicates | Same | 2 |
| **Expression columns** | Python eval → `expr` library | Same (different syntax) | 2 |
| **Undo/Redo** | Full command undo | Same | 2 |
| **Command log** | Record, replay, save `.vdj` | Same | 2 |
| **Clipboard** | Internal + system | Same | 2 |
| **Save/Export** | Multi-format export | Same | 2 |
| **Metasheets** | Columns, Sheets, Options, Commands, Errors | Same | 2 |
| **Graph/Plot** | ASCII scatter/line/bar charts | Same | 4 |
| **Canvas** | Braille/ASCII drawing | Same | 4 |
| **Menu system** | Hierarchical menus | Same | 4 |
| **Command palette** | Fuzzy search commands | Same | 4 |
| **Mouse support** | Click, scroll, drag | Same | 4 |
| **Multi-pane** | Split screen | Same | 4 |
| **Themes** | Color themes | Same | 4 |
| **Config file** | `.visidatarc` → `.vdgorc` | Same | 4 |
| **Macros** | Record/replay | Same | 4 |
| **vgit app** | Git interface | Port or skip | 5 |
| **vdsql app** | SQL IDE | Port or skip | 5 |
| **Plugin system** | Python plugins | Go plugins/Wasm/Lua | 5 |
| **API integrations** | Reddit, Zulip, Matrix, Airtable | On-demand | 5 |

### Notable Differences from Python VisiData

1. **Expression language**: Python VisiData uses `eval()` with full Python syntax. Go port will use a safe expression language (e.g., `github.com/expr-lang/expr`) with a more limited but safer syntax. Users write `price * quantity` instead of Python expressions.

2. **Plugin system**: Python's dynamic nature allows monkey-patching and runtime class modification. Go port options:
   - **Compile-time plugins**: Add loaders/features by importing packages (requires recompilation)
   - **Wasm plugins**: Load WebAssembly modules at runtime
   - **Lua scripting**: Embed Lua for user scripts (via `github.com/yuin/gopher-lua`)
   - **Yaegi Go interpreter**: Interpret Go code at runtime (via `github.com/traefik/yaegi`)

3. **No `.visidatarc` Python execution**: Replace with a structured config file (TOML/YAML) for options and keybindings. Expression columns provide the dynamic computation.

4. **Pandas integration**: Not applicable. VisiData's Pandas loaders (Stata, SAS, Feather, etc.) would need native Go implementations or be deferred to P3.

---

## 10. Risks, Trade-offs, and Decisions

### 10.1 Key Decisions

| Decision | Options | Recommendation | Rationale |
|----------|---------|----------------|-----------|
| TUI library | `tcell` vs `tview` vs `bubbletea` | **`tcell/v2`** | Most direct equivalent to curses; full control over rendering; mature |
| SQLite | `mattn/go-sqlite3` (CGo) vs `modernc.org/sqlite` (pure Go) | **`modernc.org/sqlite`** | Pure Go = single binary, no CGo dependency |
| Expression engine | `govaluate` vs `expr-lang/expr` vs custom | **`expr-lang/expr`** | Type-safe, fast, well-maintained, good syntax |
| CLI framework | `cobra` vs `pflag` vs stdlib `flag` | **`cobra`** | Industry standard; auto-completions; subcommands |
| Config format | TOML vs YAML vs JSON | **TOML** | Simple, readable, Go-friendly (`BurntSushi/toml`) |
| Plugin system | Compile-time vs Wasm vs Lua | **Compile-time + Lua** | Compile-time for performance; Lua for user scripts |

### 10.2 Risks

| Risk | Severity | Mitigation |
|------|----------|------------|
| **Expression language gap** | Medium | `expr` library covers 90% of use cases; document differences |
| **Plugin ecosystem** | High | Compile-time plugins are straightforward; Lua covers user needs |
| **Terminal compatibility** | Medium | `tcell` handles most terminals; test on common emulators |
| **Feature creep** | High | Strict phased approach; MVP first |
| **Performance regression** | Low | Go is generally faster than Python for data processing |
| **Wide character rendering** | Medium | Use `go-runewidth` + tcell's built-in support |
| **Missing format libraries** | Medium | Some niche formats (HDF5, SAS) may need CGo or be deferred |
| **Porting effort underestimated** | Medium | Start with MVP; iterate based on user feedback |

### 10.3 Trade-offs

| Aspect | Python VisiData | Go Port |
|--------|----------------|---------|
| **Startup time** | ~500ms | ~10ms ✅ |
| **Memory usage** | Higher (Python overhead) | Lower ✅ |
| **Distribution** | pip install + deps | Single binary ✅ |
| **Plugin flexibility** | Full Python runtime | Limited (Lua/Wasm/compile-time) ⚠️ |
| **Expression power** | Full Python syntax | Safe subset ⚠️ |
| **Development speed** | Faster prototyping | More boilerplate ⚠️ |
| **Type safety** | Runtime errors | Compile-time checks ✅ |
| **Concurrency** | GIL-limited threads | Goroutines ✅ |
| **Format coverage** | 70+ via pip | Depends on Go libraries ⚠️ |

---

## 11. Build, Release, and Distribution

### 11.1 Build System

```makefile
# Makefile
BINARY := vdgo
VERSION := $(shell git describe --tags --always)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build test lint release

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/vd/

test:
	go test ./...

lint:
	golangci-lint run

# Cross-compile for all platforms
release:
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64 ./cmd/vd/
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64 ./cmd/vd/
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64 ./cmd/vd/
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64 ./cmd/vd/
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-windows-amd64.exe ./cmd/vd/
```

### 11.2 Distribution Channels

| Channel | Approach |
|---------|----------|
| **GitHub Releases** | `goreleaser` for automated binary builds |
| **Homebrew** | Formula in `homebrew-core` or custom tap |
| **Snap** | `snapcraft.yaml` |
| **Docker** | `FROM scratch` with single binary |
| **APT/RPM** | `nfpm` for Linux packages |
| **Scoop** | Windows package manager |
| **Go install** | `go install github.com/user/vdgo/cmd/vd@latest` |

### 11.3 CI/CD

```yaml
# .github/workflows/ci.yml
- Build and test on Linux/macOS/Windows
- golangci-lint for code quality
- goreleaser for tagged releases
- Integration tests with sample data files
```

---

## 12. Testing Strategy

### 12.1 Unit Tests

```
internal/core/*_test.go     # Core logic: types, sort, select, aggregate
internal/loader/*_test.go   # Loader round-trip tests (load → verify → save → reload)
internal/tui/*_test.go      # TUI component tests (mock screen)
internal/feature/*_test.go  # Feature tests (join, pivot, etc.)
```

### 12.2 Integration Tests

- **Golden file tests**: Load sample data → execute commands → compare output with golden files (mirrors VisiData's `tests/golden/` approach)
- **Round-trip tests**: Load format → save → reload → verify identical
- **CLI tests**: Run binary with arguments → verify exit code and output

### 12.3 Compatibility Tests

- Compare output with Python VisiData for the same input files
- Ensure keybindings match (same keys for same operations)
- Validate command names match (`delete-row`, `sort-asc`, etc.)

### 12.4 Sample Data

Reuse VisiData's `sample_data/` directory for test fixtures.

---

## 13. Estimated Effort

| Phase | Duration | Effort | Deliverable |
|-------|----------|--------|-------------|
| **Phase 0**: Foundation | 3 weeks | 1 developer | CSV viewer with navigation |
| **Phase 1**: Core Operations | 4 weeks | 1–2 developers | Full data exploration for text formats |
| **Phase 2**: Advanced Features | 5 weeks | 1–2 developers | Joins, pivot, frequency, expression columns |
| **Phase 3**: Extended Loaders | 6 weeks | 1–2 developers | 20+ format support |
| **Phase 4**: Advanced UI | 6 weeks | 1–2 developers | Graphs, menus, multi-pane, themes |
| **Phase 5**: Apps & Ecosystem | 6 weeks | 1–2 developers | vgit, vdsql, plugins, packaging |
| **Total** | **~30 weeks** | **1–2 developers** | **Full VisiData replacement** |

**Minimum viable product (Phases 0–1)**: ~7 weeks, 1 developer.
**Practical daily-driver (Phases 0–2)**: ~12 weeks, 1–2 developers.
**Full replacement (all phases)**: ~30 weeks, 1–2 developers.

---

## Appendix A: Key VisiData Commands to Port

The following is a prioritized list of the most commonly used VisiData commands that should be ported first:

### Navigation (Phase 0)
| Key | Command | Description |
|-----|---------|-------------|
| `h/j/k/l` | cursor-left/down/up/right | Move cursor |
| `gj/gk` | go-bottom/go-top | Jump to first/last row |
| `gh/gl` | go-leftmost/go-rightmost | Jump to first/last column |
| `PgDn/PgUp` | page-down/page-up | Page scroll |
| `q` | quit-sheet | Close current sheet |
| `Ctrl+Q` | quit-all | Quit VisiData |

### Data Types (Phase 1)
| Key | Command | Description |
|-----|---------|-------------|
| `~` | type-string | Set column type to string |
| `#` | type-int | Set column type to integer |
| `%` | type-float | Set column type to float |
| `$` | type-currency | Set column type to currency |
| `@` | type-date | Set column type to date |

### Column Operations (Phase 1)
| Key | Command | Description |
|-----|---------|-------------|
| `_` | resize-col-max | Resize column to fit content |
| `-` | hide-col | Hide current column |
| `^` | rename-col | Rename column |
| `!` | key-col | Toggle key column |
| `=` | addcol-expr | Add expression column |

### Sorting & Selection (Phase 1)
| Key | Command | Description |
|-----|---------|-------------|
| `[` | sort-asc | Sort ascending by current column |
| `]` | sort-desc | Sort descending by current column |
| `s` | select-row | Select current row |
| `u` | unselect-row | Unselect current row |
| `t` | toggle-row | Toggle row selection |
| `gs` | select-all | Select all rows |
| `gu` | unselect-all | Unselect all rows |
| `\|` | select-col-regex | Select rows matching regex |
| `\\` | unselect-col-regex | Unselect rows matching regex |

### Search (Phase 1)
| Key | Command | Description |
|-----|---------|-------------|
| `/` | search-col | Search current column |
| `n` | search-next | Next search match |
| `N` | search-prev | Previous search match |

### Sheet Operations (Phase 2)
| Key | Command | Description |
|-----|---------|-------------|
| `F` | freq-col | Frequency table for column |
| `I` | describe-all | Statistical summary |
| `W` | pivot | Pivot table |
| `T` | transpose | Transpose sheet |
| `M` | melt | Unpivot/melt |
| `&` | join-sheets | Join selected sheets |
| `C` | columns-sheet | Open columns sheet |
| `S` | sheets-sheet | Open sheets sheet |
| `O` | options-sheet | Open options sheet |
| `Ctrl+S` | save-sheet | Save current sheet |
| `d` | delete-row | Delete current row |
| `e` | edit-cell | Edit cell value |
| `z=` | setcol-expr | Set column values from expression |

---

## Appendix B: Expression Language Comparison

Python VisiData expressions use full Python syntax. The Go port should use a safe, familiar expression syntax:

| Operation | Python VisiData | Go Port (`expr` library) |
|-----------|----------------|--------------------------|
| Column reference | `price` | `price` |
| Arithmetic | `price * quantity` | `price * quantity` |
| String concat | `first + " " + last` | `first + " " + last` |
| Conditional | `"yes" if price > 10 else "no"` | `price > 10 ? "yes" : "no"` |
| String method | `name.upper()` | `upper(name)` |
| Regex match | `re.search(r"\d+", col1)` | `matches("\\d+", col1)` |
| Math functions | `math.sqrt(x)` | `sqrt(x)` |
| Null check | `col1 if col1 else "N/A"` | `col1 ?? "N/A"` |
| List/array | `len(items)` | `len(items)` |
| Type cast | `int(price)` | `int(price)` |

---

## Appendix C: Go Library Version Recommendations

```go
// go.mod (recommended starting dependencies)
module github.com/user/vdgo

go 1.22

require (
    // TUI
    github.com/gdamore/tcell/v2    v2.7+
    github.com/mattn/go-runewidth   v0.0.16+

    // CLI
    github.com/spf13/cobra          v1.8+

    // Data formats
    github.com/xuri/excelize/v2     v2.8+
    gopkg.in/yaml.v3                v3.0+
    github.com/BurntSushi/toml      v1.4+
    github.com/parquet-go/parquet-go v0.23+
    modernc.org/sqlite              v1.34+

    // Database
    github.com/jackc/pgx/v5         v5.7+
    github.com/go-sql-driver/mysql  v1.8+

    // Expression engine
    github.com/expr-lang/expr       v1.16+

    // Utilities
    github.com/araddon/dateparse    v0.0.0-latest
    github.com/dustin/go-humanize   v1.0+
    github.com/sahilm/fuzzy         v0.1+
    github.com/klauspost/compress   v1.17+
    golang.org/x/text               v0.21+

    // AWS (optional, for S3)
    github.com/aws/aws-sdk-go-v2    v1.34+
)
```

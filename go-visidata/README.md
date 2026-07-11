# go-visidata

Initial Go implementation slice for VisiData.

Current scope:
- load CSV/TSV/PSV-style delimited files
- load XLSX workbooks
- load JSON and JSONL/NDJSON inputs
- load YAML, TOML, XML, Markdown tables, and HTML tables
- load fixed-width text
- load directory listings
- load ZIP and TAR archives as entry listings
- open HTTP/HTTPS URLs and compressed inputs transparently
- load SQLite databases
- infer simple column types (`string`, `int`, `float`, `bool`, `date`)
- add materialized expression columns from existing row values
- keep a basic in-memory undo stack for sheet mutations
- render a terminal preview table
- open a minimal interactive terminal viewer with basic navigation
- open derived analysis sheets for frequency counts, describe summaries, and transpose views
- open melt/unpivot views using the current column as an identifier
- record an in-memory command log for the current interactive session

## Usage

```sh
go run ./cmd/vdgo -- ../sample_data/benchmark.csv
go run ./cmd/vdgo -- ./fixtures/people.xlsx
go run ./cmd/vdgo -- ../sample_data/benchmark.jsonl
go run ./cmd/vdgo -- ../sample_data/test.fixed
go run ./cmd/vdgo -filetype fixed -header 0 -- ../sample_data/test-fixed-leadingspaces.txt
go run ./cmd/vdgo -- ..
go run ./cmd/vdgo -- ../tests/without_rowid.db
go run ./cmd/vdgo -table withrowid -- ../tests/without_rowid.db
go run ./cmd/vdgo -- ./fixtures/people.yaml
go run ./cmd/vdgo -- https://example.com/people.csv.gz
go run ./cmd/vdgo -n 5 -- ../sample_data/a.tsv
go run ./cmd/vdgo -save /tmp/benchmark.json -- ../sample_data/benchmark.jsonl
go run ./cmd/vdgo -save /tmp/benchmark.yaml -- ../sample_data/benchmark.jsonl
go run ./cmd/vdgo -config ~/.vdgorc -- ../tests/without_rowid.db
printf '{"name":"Alice","age":30}\n{"name":"Bob","age":40}\n' | go run ./cmd/vdgo -n 2 -- -
```

## Build

For portable Linux binaries, build with cgo disabled so the executable is statically linked and does not depend on the target machine's glibc version:

```sh
CGO_ENABLED=0 go build -o /tmp/vdgo ./cmd/vdgo
```

From the repository root, `make go-build` now does this by default and writes `/tmp/vdgo`. Override it when needed:

```sh
make go-build
make go-build GO_BUILD_OUTPUT=./vdgo
make go-build CGO_ENABLED=1
```

`make go-test` also uses the same `CGO_ENABLED` setting, so `make go-test` validates the portable build path by default.

## Controls

- `q`: quit, or close the current derived sheet and return to the previous sheet
- `Esc`/`Ctrl+C`: quit
- `Enter`: open the current row as a column/value table, or open the matching subset from a frequency sheet
- arrow keys or `hjkl`: move cursor
- `PageUp`/`PageDown`: move by page
- `Home`/`g`: first row
- `End`/`G`: last row
- `[` / `]`: sort ascending / descending on the current column
- `=`: add an expression column; use `name := expr` to name it explicitly
- `&`: join with the previous sheet in the stack using the current column
- `f`: open a frozen snapshot of the current sheet
- `v`: open a graph for the current column; numeric pairs open a scatter plot, a numeric column alone opens a line chart, and a label column with a numeric partner opens a bar chart
- `*`: add a sparkline column from the visible numeric columns in each row
- `m`: open a melt/unpivot view using the current column as the identifier
- `D`: open a deduplicated view using the current column
- `F`: open a frequency table for the current column; press `Enter` on a value to open matching rows
- `I`: open a describe sheet for the current sheet
- `W`: open a pivot view grouped by the current column
- `T`: transpose the current sheet
- `V`: open the current sheet stack and press `Enter` to jump to a sheet
- `M`: open an editable columns metasheet for the current sheet
- `O`: open a viewer state sheet for the current sheet
- `P`: open the in-memory command log for the current session
- `?`: open the command reference sheet
- `Tab`: toggle the sheet-stack sidebar; use arrow keys and `Enter` to switch sheets
- `:` or `Ctrl+P`: open the command palette with fuzzy search
- `;` or `F10`: open the menu bar and navigate it with arrows and `Enter`
- `c`: search visible column names by regex and jump to the next matching column
- `r`: jump to the next row whose visible row key matches a regex
- `zc` / `zr`: jump to a 0-based visible column number or row number
- `zz`: start or stop recording a macro, `zZ`: replay the last recorded macro
- mouse click: move the cursor to a cell
- mouse wheel: scroll rows
- drag with button 1: select a row range
- `R`: redo the last undone sheet mutation on the current sheet
- `U`: undo the last sheet mutation on the current sheet
- `e`: edit the current cell in place
- `/`: search for a case-insensitive substring, `n` / `N`: next / previous match
- `s`: toggle current row selection, `t`: select all rows, `u`: clear selection
- `|`: regex-select rows matching the current column, `\`: regex-unselect matching rows
- `y`: copy the current cell to the internal clipboard
- `C`: copy selected rows (or the current row) to the internal clipboard
- `d`: delete selected rows (or the current row) and copy the deleted data to the clipboard
- `S` / `Ctrl+S`: save the current sheet to a suggested export path
- `-`: hide the current column, `H`: show all hidden columns
- `^`: rename the current column
- `_`: set the display width for the current column
- `{`: add regex capture-group columns from the current column
- `}`: split the current column into multiple columns using a regex
- `~` / `#` / `%` / `$` / `@`: override the current column type to string, int, float, currency, or date

Expression columns can reference simple column names directly and always expose `col1`, `col2`, ... aliases for the table columns. Use `str(...)` when concatenating non-string values. For example: `total := qty * price` or `str(col1 * 10) + "-" + City`.

Use `-n` to keep the existing non-interactive preview mode.
Use `-save` to export the loaded sheet to a `.csv`, `.tsv`, `.json`, `.yaml`, `.toml`, or `.xlsx` file and exit.
Use `-filetype fixed` to force the fixed-width loader on plain text files.
Use `-header 0` to keep the first fixed-width row as data instead of column names.
Use `-table` to choose a specific SQLite table or XLSX sheet.
Use `-theme` to pick a built-in interactive theme: `default`, `amber`, or `ocean`.
Use `-show-hidden` to include dotfiles in directory listings.
Use `-config` to load a TOML config file explicitly. Without `-config`, `vdgo` looks for `.vdgorc` in the current directory first, then `~/.vdgorc`, and `VDGO_CONFIG` overrides both.

Example `.vdgorc`:

```toml
preview_rows = 5
filetype = "fixed"
header = 0
table = "Sheet2"
theme = "amber"
show_hidden = true
```

Config values fill in defaults for `filetype`, `header`, `preview_rows`, `table`, `theme`, and `show_hidden`. Explicit CLI flags still take precedence.

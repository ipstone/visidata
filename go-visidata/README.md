# go-visidata

Initial Go implementation slice for VisiData.

Current scope:
- load CSV/TSV/PSV-style delimited files
- load JSON and JSONL/NDJSON inputs
- load fixed-width text
- load directory listings
- load SQLite databases
- infer simple column types (`string`, `int`, `float`, `bool`, `date`)
- render a terminal preview table
- open a minimal interactive terminal viewer with basic navigation
- open derived analysis sheets for frequency counts, describe summaries, and transpose views

## Usage

```sh
go run ./cmd/vdgo -- ../sample_data/benchmark.csv
go run ./cmd/vdgo -- ../sample_data/benchmark.jsonl
go run ./cmd/vdgo -- ../sample_data/test.fixed
go run ./cmd/vdgo -filetype fixed -header 0 -- ../sample_data/test-fixed-leadingspaces.txt
go run ./cmd/vdgo -- ..
go run ./cmd/vdgo -- ../tests/without_rowid.db
go run ./cmd/vdgo -table withrowid -- ../tests/without_rowid.db
go run ./cmd/vdgo -n 5 -- ../sample_data/a.tsv
go run ./cmd/vdgo -save /tmp/benchmark.json -- ../sample_data/benchmark.jsonl
printf '{"name":"Alice","age":30}\n{"name":"Bob","age":40}\n' | go run ./cmd/vdgo -n 2 -- -
```

## Controls

- `q`: quit, or close the current derived sheet and return to the previous sheet
- `Esc`/`Ctrl+C`: quit
- arrow keys or `hjkl`: move cursor
- `PageUp`/`PageDown`: move by page
- `Home`/`g`: first row
- `End`/`G`: last row
- `[` / `]`: sort ascending / descending on the current column
- `&`: join with the previous sheet in the stack using the current column
- `f`: open a frozen snapshot of the current sheet
- `D`: open a deduplicated view using the current column
- `F`: open a frequency table for the current column
- `I`: open a describe sheet for the current sheet
- `W`: open a pivot view grouped by the current column
- `T`: transpose the current sheet
- `V`: open the current sheet stack and press `Enter` to jump to a sheet
- `M`: open an editable columns metasheet for the current sheet
- `O`: open a viewer state sheet for the current sheet
- `?`: open the command reference sheet
- `/`: search for a case-insensitive substring, `n` / `N`: next / previous match
- `s`: toggle current row selection, `t`: select all rows, `u`: clear selection
- `|`: regex-select rows matching the current column, `\`: regex-unselect matching rows
- `c`: copy the current cell to the internal clipboard
- `C`: copy selected rows (or the current row) to the internal clipboard
- `d`: delete selected rows (or the current row) and copy the deleted data to the clipboard
- `S` / `Ctrl+S`: save the current sheet to a suggested export path
- `-`: hide the current column, `H`: show all hidden columns
- `^`: rename the current column
- `_`: set the display width for the current column
- `~` / `#` / `%` / `$` / `@`: override the current column type to string, int, float, currency, or date

Use `-n` to keep the existing non-interactive preview mode.
Use `-save` to export the loaded sheet to a `.csv`, `.tsv`, or `.json` file and exit.
Use `-filetype fixed` to force the fixed-width loader on plain text files.
Use `-header 0` to keep the first fixed-width row as data instead of column names.
Use `-table` to choose a specific SQLite table.

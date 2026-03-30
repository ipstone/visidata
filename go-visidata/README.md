# go-visidata

Initial Go implementation slice for VisiData.

Current scope:
- load CSV/TSV/PSV-style delimited files
- load JSON and JSONL/NDJSON inputs
- infer simple column types (`string`, `int`, `float`, `bool`, `date`)
- render a terminal preview table
- open a minimal interactive terminal viewer with basic navigation

## Usage

```sh
go run ./cmd/vdgo -- ../sample_data/benchmark.csv
go run ./cmd/vdgo -- ../sample_data/benchmark.jsonl
go run ./cmd/vdgo -n 5 -- ../sample_data/a.tsv
printf '{"name":"Alice","age":30}\n{"name":"Bob","age":40}\n' | go run ./cmd/vdgo -n 2 -- -
```

## Controls

- `q`/`Esc`/`Ctrl+C`: quit
- arrow keys or `hjkl`: move cursor
- `PageUp`/`PageDown`: move by page
- `Home`/`g`: first row
- `End`/`G`: last row

Use `-n` to keep the existing non-interactive preview mode.

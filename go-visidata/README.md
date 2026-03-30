# go-visidata

Initial Go implementation slice for VisiData.

Current scope:
- load CSV/TSV/PSV-style delimited files
- infer simple column types (`string`, `int`, `float`, `bool`, `date`)
- render a terminal preview table

## Usage

```sh
go run ./cmd/vdgo -- ../sample_data/benchmark.csv
go run ./cmd/vdgo -n 5 -- ../sample_data/a.tsv
```

package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ipstone/visidata/go-visidata/pkg/sheet"
)

type format string

const (
	formatDelimited format = "delimited"
	formatDir       format = "dir"
	formatFixed     format = "fixed"
	formatJSON      format = "json"
	formatJSONL     format = "jsonl"
	formatSQLite    format = "sqlite"
)

type Options struct {
	Format     string
	Header     int
	ShowHidden bool
	Table      string
}

func Load(path string) (*sheet.Sheet, error) {
	return LoadWithOptions(path, Options{Header: 1})
}

func LoadWithOptions(path string, opts Options) (*sheet.Sheet, error) {
	if path == "" {
		return nil, fmt.Errorf("no input path provided")
	}

	info, err := os.Stat(path)
	if err == nil && info.IsDir() {
		return loadDir(path, opts)
	}

	reader, closeFn, err := openReader(path)
	if err != nil {
		return nil, err
	}
	defer closeFn()

	fileFormat := format("")
	if opts.Format != "" {
		fileFormat = format(strings.ToLower(opts.Format))
	} else {
		fileFormat, err = detectFormat(path, reader)
		if err != nil {
			return nil, err
		}
	}

	switch fileFormat {
	case formatDir:
		return loadDir(path, opts)
	case formatFixed:
		return loadFixed(path, reader, opts)
	case formatSQLite:
		return loadSQLite(path, opts)
	case formatJSON, formatJSONL:
		return loadJSON(path, reader, fileFormat)
	default:
		return loadDelimited(path, reader)
	}
}

func detectFormat(path string, reader *os.File) (format, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".fixed":
		return formatFixed, nil
	case ".json":
		return formatJSON, nil
	case ".jsonl", ".ndjson", ".ldjson":
		return formatJSONL, nil
	case ".csv", ".tsv", ".psv":
		return formatDelimited, nil
	case ".db", ".sqlite", ".sqlite3":
		return formatSQLite, nil
	}

	line, err := firstDataLine(reader)
	if err != nil {
		return "", err
	}

	switch {
	case strings.HasPrefix(line, "["):
		return formatJSON, nil
	case strings.HasPrefix(line, "{"):
		return formatJSONL, nil
	default:
		return formatDelimited, nil
	}
}

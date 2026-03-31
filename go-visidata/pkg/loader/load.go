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
	formatHTML      format = "html"
	formatJSON      format = "json"
	formatJSONL     format = "jsonl"
	formatMarkdown  format = "markdown"
	formatArchive   format = "archive"
	formatSQLite    format = "sqlite"
	formatTOML      format = "toml"
	formatXML       format = "xml"
	formatXLSX      format = "xlsx"
	formatYAML      format = "yaml"
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
	case formatArchive:
		return loadArchive(path, reader)
	case formatFixed:
		return loadFixed(path, reader, opts)
	case formatHTML:
		return loadHTML(path, reader)
	case formatMarkdown:
		return loadMarkdown(path, reader)
	case formatSQLite:
		return loadSQLite(path, opts)
	case formatTOML:
		return loadTOML(path, reader)
	case formatXML:
		return loadXML(path, reader)
	case formatXLSX:
		return loadXLSX(path, reader, opts)
	case formatYAML:
		return loadYAML(path, reader)
	case formatJSON, formatJSONL:
		return loadJSON(path, reader, fileFormat)
	default:
		return loadDelimited(path, reader)
	}
}

func detectFormat(path string, reader *os.File) (format, error) {
	switch normalizedExt(path) {
	case ".zip", ".tar":
		return formatArchive, nil
	case ".fixed":
		return formatFixed, nil
	case ".htm", ".html":
		return formatHTML, nil
	case ".json":
		return formatJSON, nil
	case ".jsonl", ".ndjson", ".ldjson":
		return formatJSONL, nil
	case ".md", ".markdown":
		return formatMarkdown, nil
	case ".csv", ".tsv", ".psv":
		return formatDelimited, nil
	case ".db", ".sqlite", ".sqlite3":
		return formatSQLite, nil
	case ".toml":
		return formatTOML, nil
	case ".xml":
		return formatXML, nil
	case ".xlsx":
		return formatXLSX, nil
	case ".yaml", ".yml":
		return formatYAML, nil
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
	case strings.HasPrefix(line, "<"):
		return formatXML, nil
	default:
		return formatDelimited, nil
	}
}

func normalizedExt(path string) string {
	effective := stripCompressionSuffix(path)
	lower := strings.ToLower(effective)
	if strings.HasSuffix(lower, ".tar") {
		return ".tar"
	}
	return strings.ToLower(filepath.Ext(effective))
}

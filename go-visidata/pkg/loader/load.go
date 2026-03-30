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
	formatJSON      format = "json"
	formatJSONL     format = "jsonl"
)

func Load(path string) (*sheet.Sheet, error) {
	if path == "" {
		return nil, fmt.Errorf("no input path provided")
	}

	reader, closeFn, err := openReader(path)
	if err != nil {
		return nil, err
	}
	defer closeFn()

	fileFormat, err := detectFormat(path, reader)
	if err != nil {
		return nil, err
	}

	switch fileFormat {
	case formatJSON, formatJSONL:
		return loadJSON(path, reader, fileFormat)
	default:
		return loadDelimited(path, reader)
	}
}

func detectFormat(path string, reader *os.File) (format, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return formatJSON, nil
	case ".jsonl", ".ndjson", ".ldjson":
		return formatJSONL, nil
	case ".csv", ".tsv", ".psv":
		return formatDelimited, nil
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

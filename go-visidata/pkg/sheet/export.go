package sheet

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func (s *Sheet) Export(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create export file: %w", err)
	}
	defer file.Close()

	switch strings.ToLower(filepath.Ext(path)) {
	case ".tsv":
		return s.writeDelimited(file, '\t')
	case ".json":
		return s.writeJSON(file)
	default:
		return s.writeDelimited(file, ',')
	}
}

func (s *Sheet) SaveSuggested() (string, error) {
	path := s.SuggestedSavePath()
	if err := s.Export(path); err != nil {
		return "", err
	}
	return path, nil
}

func (s *Sheet) SuggestedSavePath() string {
	dir := "."
	source := s.Source
	if source != "" && source != "-" {
		dir = filepath.Dir(source)
	}

	base := s.Name
	if source != "" && source != "-" {
		if info, err := os.Stat(source); err == nil && info.IsDir() {
			base = filepath.Base(source)
		} else {
			base = strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
		}
	}
	if base == "" || base == "." || base == string(filepath.Separator) {
		base = "stdin"
	}
	base = strings.ReplaceAll(base, ":", "_")
	return filepath.Join(dir, base+".vdgo"+s.suggestedExtension())
}

func (s *Sheet) suggestedExtension() string {
	switch strings.ToLower(filepath.Ext(s.Source)) {
	case ".tsv":
		return ".tsv"
	case ".json", ".jsonl", ".ndjson", ".ldjson":
		return ".json"
	default:
		return ".csv"
	}
}

func (s *Sheet) writeDelimited(file *os.File, comma rune) error {
	writer := csv.NewWriter(file)
	writer.Comma = comma

	header := make([]string, len(s.Columns))
	for i, col := range s.Columns {
		header[i] = col.Name
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	for _, row := range s.Rows {
		record := make([]string, len(s.Columns))
		copy(record, row)
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("write row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("flush export: %w", err)
	}
	return nil
}

func (s *Sheet) writeJSON(file *os.File) error {
	rows := make([]map[string]any, 0, len(s.Rows))
	for _, row := range s.Rows {
		record := make(map[string]any, len(s.Columns))
		for colIndex, col := range s.Columns {
			record[col.Name] = typedValue(col.Kind, cellAt(row, colIndex))
		}
		rows = append(rows, record)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(rows); err != nil {
		return fmt.Errorf("encode json export: %w", err)
	}
	return nil
}

func typedValue(kind ValueKind, value string) any {
	switch kind {
	case KindInt:
		if parsed, err := strconv.ParseInt(normalizeNumber(value), 10, 64); err == nil {
			return parsed
		}
	case KindFloat:
		if parsed, err := strconv.ParseFloat(normalizeNumber(value), 64); err == nil {
			return parsed
		}
	case KindBool:
		if parsed, err := strconv.ParseBool(strings.ToLower(value)); err == nil {
			return parsed
		}
	}
	return value
}

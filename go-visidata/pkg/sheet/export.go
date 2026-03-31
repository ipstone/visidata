package sheet

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/xuri/excelize/v2"
	"gopkg.in/yaml.v3"
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
	case ".toml":
		return s.writeTOML(file)
	case ".xlsx":
		return s.writeXLSX(file)
	case ".yaml", ".yml":
		return s.writeYAML(file)
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
	case ".toml":
		return ".toml"
	case ".xlsx":
		return ".xlsx"
	case ".yaml", ".yml":
		return ".yaml"
	default:
		return ".csv"
	}
}

func (s *Sheet) writeDelimited(file *os.File, comma rune) error {
	writer := csv.NewWriter(file)
	writer.Comma = comma

	header, rows := s.exportTableStrings()
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
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
	rows := s.exportRows()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(rows); err != nil {
		return fmt.Errorf("encode json export: %w", err)
	}
	return nil
}

func (s *Sheet) writeYAML(file *os.File) error {
	encoder := yaml.NewEncoder(file)
	defer encoder.Close()
	if err := encoder.Encode(s.exportRows()); err != nil {
		return fmt.Errorf("encode yaml export: %w", err)
	}
	return nil
}

func (s *Sheet) writeTOML(file *os.File) error {
	payload := map[string]any{"rows": s.exportRows()}
	if err := toml.NewEncoder(file).Encode(payload); err != nil {
		return fmt.Errorf("encode toml export: %w", err)
	}
	return nil
}

func (s *Sheet) writeXLSX(file *os.File) error {
	workbook := excelize.NewFile()
	defer workbook.Close()

	sheetName := workbook.GetSheetName(0)
	header, rows := s.exportTableTyped()
	for rowIndex, row := range append([][]any{header}, rows...) {
		for colIndex, value := range row {
			cell, err := excelize.CoordinatesToCellName(colIndex+1, rowIndex+1)
			if err != nil {
				return fmt.Errorf("xlsx cell coordinates: %w", err)
			}
			if err := workbook.SetCellValue(sheetName, cell, value); err != nil {
				return fmt.Errorf("xlsx write cell %s: %w", cell, err)
			}
		}
	}

	if err := workbook.Write(file); err != nil {
		return fmt.Errorf("write xlsx export: %w", err)
	}
	return nil
}

func (s *Sheet) exportRows() []map[string]any {
	rows := make([]map[string]any, 0, len(s.Rows))
	cols := s.VisibleColumnIndices()
	for _, row := range s.Rows {
		record := make(map[string]any, len(cols))
		for _, colIndex := range cols {
			col := s.Columns[colIndex]
			record[col.Name] = typedValue(col.EffectiveKind(), cellAt(row, colIndex))
		}
		rows = append(rows, record)
	}
	return rows
}

func (s *Sheet) exportTableStrings() ([]string, [][]string) {
	cols := s.VisibleColumnIndices()
	header := make([]string, len(cols))
	rows := make([][]string, 0, len(s.Rows))
	for i, colIndex := range cols {
		header[i] = s.Columns[colIndex].Name
	}
	for _, row := range s.Rows {
		rows = append(rows, visibleRowValues(row, cols))
	}
	return header, rows
}

func (s *Sheet) exportTableTyped() ([]any, [][]any) {
	cols := s.VisibleColumnIndices()
	header := make([]any, len(cols))
	rows := make([][]any, 0, len(s.Rows))
	for i, colIndex := range cols {
		header[i] = s.Columns[colIndex].Name
	}
	for _, row := range s.Rows {
		record := make([]any, len(cols))
		for i, colIndex := range cols {
			record[i] = typedValue(s.Columns[colIndex].EffectiveKind(), cellAt(row, colIndex))
		}
		rows = append(rows, record)
	}
	return header, rows
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
	case KindCurrency:
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

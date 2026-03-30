package sheet

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ValueKind string

const (
	KindString ValueKind = "string"
	KindInt    ValueKind = "int"
	KindFloat  ValueKind = "float"
	KindBool   ValueKind = "bool"
	KindDate   ValueKind = "date"
)

type Column struct {
	Name string
	Kind ValueKind
}

type Row []string

type Sheet struct {
	Name    string
	Source  string
	Columns []Column
	Rows    []Row
}

func New(name, source string, headers []string) *Sheet {
	columns := make([]Column, len(headers))
	for i, header := range headers {
		columns[i] = Column{Name: strings.TrimSpace(header), Kind: KindString}
	}

	return &Sheet{
		Name:    baseName(name),
		Source:  source,
		Columns: columns,
	}
}

func baseName(name string) string {
	if name == "" || name == "-" {
		return "stdin"
	}

	return filepath.Base(name)
}

func (s *Sheet) AddRow(values []string) {
	row := make(Row, len(s.Columns))
	copy(row, values)
	for len(row) < len(s.Columns) {
		row = append(row, "")
	}
	if len(row) > len(s.Columns) {
		row = row[:len(s.Columns)]
	}
	for i := range row {
		row[i] = strings.TrimSpace(row[i])
	}

	s.Rows = append(s.Rows, row)
}

func (s *Sheet) InferColumnKinds() {
	for i := range s.Columns {
		s.Columns[i].Kind = inferKindForColumn(s.Rows, i)
	}
}

func (s *Sheet) Summary() string {
	return fmt.Sprintf("%s: %d row(s) x %d column(s)", s.Name, len(s.Rows), len(s.Columns))
}

func inferKindForColumn(rows []Row, columnIndex int) ValueKind {
	seen := 0
	allInt := true
	allFloat := true
	allBool := true
	allDate := true

	for _, row := range rows {
		if columnIndex >= len(row) {
			continue
		}
		value := strings.TrimSpace(row[columnIndex])
		if value == "" {
			continue
		}
		seen++
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			allInt = false
		}
		if _, err := strconv.ParseFloat(normalizeNumber(value), 64); err != nil {
			allFloat = false
		}
		if _, err := strconv.ParseBool(strings.ToLower(value)); err != nil {
			allBool = false
		}
		if !looksLikeDate(value) {
			allDate = false
		}
	}

	if seen == 0 {
		return KindString
	}
	if allInt {
		return KindInt
	}
	if allFloat {
		return KindFloat
	}
	if allBool {
		return KindBool
	}
	if allDate {
		return KindDate
	}
	return KindString
}

func normalizeNumber(value string) string {
	replacer := strings.NewReplacer(
		",", "",
		"$", "",
		"€", "",
		"£", "",
		"¥", "",
	)
	return replacer.Replace(value)
}

func looksLikeDate(value string) bool {
	// Go time layouts use the reference timestamp 1/2/2006 3:04PM.
	// These layouts cover the simple ISO and month/day formats seen in the
	// existing VisiData sample data fixtures.
	layouts := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02 15:04:05",
		"1/2/2006",
		"1/2/2006 3:04PM",
		"1/2/2006 3:04pm",
		"1/2/2006 3:04p",
		"1/2/2006 3:04 PM",
		"1/2/2006 3:04 pm",
	}

	for _, layout := range layouts {
		if _, err := time.Parse(layout, value); err == nil {
			return true
		}
	}

	return false
}

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
	Name      string
	Source    string
	Columns   []Column
	Rows      []Row
	CursorRow int
	CursorCol int
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

func (s *Sheet) Cell(row, col int) string {
	if row < 0 || row >= len(s.Rows) || col < 0 || col >= len(s.Columns) {
		return ""
	}
	if col >= len(s.Rows[row]) {
		return ""
	}
	return s.Rows[row][col]
}

func (s *Sheet) MoveCursorRow(delta int) {
	s.CursorRow += delta
	s.clampCursor()
}

func (s *Sheet) MoveCursorCol(delta int) {
	s.CursorCol += delta
	s.clampCursor()
}

func (s *Sheet) SetCursorRow(row int) {
	s.CursorRow = row
	s.clampCursor()
}

func (s *Sheet) SetCursorCol(col int) {
	s.CursorCol = col
	s.clampCursor()
}

func (s *Sheet) clampCursor() {
	if len(s.Rows) == 0 {
		s.CursorRow = 0
	} else {
		if s.CursorRow < 0 {
			s.CursorRow = 0
		}
		if s.CursorRow >= len(s.Rows) {
			s.CursorRow = len(s.Rows) - 1
		}
	}

	if len(s.Columns) == 0 {
		s.CursorCol = 0
		return
	}
	if s.CursorCol < 0 {
		s.CursorCol = 0
	}
	if s.CursorCol >= len(s.Columns) {
		s.CursorCol = len(s.Columns) - 1
	}
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
	// These layouts cover the simple ISO and month/day formats seen in the
	// existing VisiData sample data fixtures.
	layouts := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02 15:04:05",
		// Go layouts are case-sensitive, so these variants match fixture suffixes
		// like `PM`, `pm`, and the short `p`.
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

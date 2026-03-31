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
	KindString   ValueKind = "string"
	KindInt      ValueKind = "int"
	KindFloat    ValueKind = "float"
	KindBool     ValueKind = "bool"
	KindDate     ValueKind = "date"
	KindCurrency ValueKind = "currency"
)

var dateLayouts = []string{
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

type Column struct {
	Name         string
	Kind         ValueKind
	OverrideKind ValueKind
	Hidden       bool
	Width        int
}

type Row []string

type SortDirection string

const (
	SortNone SortDirection = ""
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

type SortState struct {
	ColumnIndex int
	Direction   SortDirection
}

type Position struct {
	Row int
	Col int
}

type SearchState struct {
	Query        string
	Matches      []Position
	CurrentMatch int
}

type Clipboard struct {
	Content string
	Kind    string
	Count   int
}

type GraphKind string

const (
	GraphBar     GraphKind = "bar"
	GraphLine    GraphKind = "line"
	GraphScatter GraphKind = "scatter"
)

type GraphPoint struct {
	Label string
	X     float64
	Y     float64
}

type GraphSpec struct {
	Kind   GraphKind
	Title  string
	XLabel string
	YLabel string
	Points []GraphPoint
}

type Sheet struct {
	Name        string
	Source      string
	Columns     []Column
	Rows        []Row
	MetaKind    string
	MetaCols    []int
	MetaRows    []int
	MetaTargets []*Sheet
	CursorRow   int
	CursorCol   int
	Selected    []bool
	SortState   SortState
	SearchState SearchState
	Clipboard   Clipboard
	Status      string
	Graph       *GraphSpec
	rowIDs      []int
	nextRowID   int
	undoStack   []sheetUndoEntry
	redoStack   []sheetUndoEntry
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
	s.addRow(values, true)
}

func (s *Sheet) AddRawRow(values []string) {
	s.addRow(values, false)
}

func (s *Sheet) addRow(values []string, trim bool) {
	row := make(Row, len(s.Columns))
	copy(row, values)
	for len(row) < len(s.Columns) {
		row = append(row, "")
	}
	if len(row) > len(s.Columns) {
		row = row[:len(s.Columns)]
	}
	if trim {
		for i := range row {
			row[i] = strings.TrimSpace(row[i])
		}
	}

	s.Rows = append(s.Rows, row)
	s.Selected = append(s.Selected, false)
	s.rowIDs = append(s.rowIDs, s.nextRowID)
	s.nextRowID++
}

func (s *Sheet) InferColumnKinds() {
	for i := range s.Columns {
		s.Columns[i].Kind = inferKindForColumn(s.Rows, i)
	}
}

func (s *Sheet) Summary() string {
	if s.Graph != nil {
		return fmt.Sprintf("%s: %d point(s) [%s]", s.Name, len(s.Rows), s.Graph.Kind)
	}
	if hidden := s.HiddenColumnCount(); hidden > 0 {
		return fmt.Sprintf("%s: %d row(s) x %d/%d column(s)", s.Name, len(s.Rows), len(s.VisibleColumnIndices()), len(s.Columns))
	}
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
	if delta == 0 || len(s.Columns) == 0 {
		s.clampCursor()
		return
	}

	s.clampCursor()
	step := 1
	if delta < 0 {
		step = -1
	}
	for moves := 0; moves < abs(delta); moves++ {
		next := s.CursorCol
		for {
			next += step
			if next < 0 || next >= len(s.Columns) {
				s.clampCursor()
				return
			}
			if !s.Columns[next].Hidden {
				s.CursorCol = next
				break
			}
		}
	}
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
	if len(s.VisibleColumnIndices()) == 0 {
		s.CursorCol = 0
		return
	}
	if s.Columns[s.CursorCol].Hidden {
		if visible := s.nextVisibleColumn(s.CursorCol, 1); visible >= 0 {
			s.CursorCol = visible
			return
		}
		if visible := s.nextVisibleColumn(s.CursorCol, -1); visible >= 0 {
			s.CursorCol = visible
		}
	}
}

func (s *Sheet) IsSelected(row int) bool {
	return row >= 0 && row < len(s.Selected) && s.Selected[row]
}

func (s *Sheet) ToggleSelected(row int) {
	if row < 0 || row >= len(s.Selected) {
		return
	}
	s.Selected[row] = !s.Selected[row]
}

func (s *Sheet) SelectAll() {
	for i := range s.Selected {
		s.Selected[i] = true
	}
}

func (s *Sheet) ClearSelection() {
	for i := range s.Selected {
		s.Selected[i] = false
	}
}

func (s *Sheet) SelectedCount() int {
	count := 0
	for _, selected := range s.Selected {
		if selected {
			count++
		}
	}
	return count
}

func (c Column) EffectiveKind() ValueKind {
	if c.OverrideKind != "" {
		return c.OverrideKind
	}
	return c.Kind
}

func (c Column) Label() string {
	return fmt.Sprintf("%s <%s>", c.Name, c.EffectiveKind())
}

func (s *Sheet) VisibleColumnIndices() []int {
	cols := make([]int, 0, len(s.Columns))
	for i, col := range s.Columns {
		if !col.Hidden {
			cols = append(cols, i)
		}
	}
	return cols
}

func (s *Sheet) HiddenColumnCount() int {
	count := 0
	for _, col := range s.Columns {
		if col.Hidden {
			count++
		}
	}
	return count
}

func (s *Sheet) nextVisibleColumn(start, delta int) int {
	for i := start + delta; i >= 0 && i < len(s.Columns); i += delta {
		if !s.Columns[i].Hidden {
			return i
		}
	}
	return -1
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func inferKindForColumn(rows []Row, columnIndex int) ValueKind {
	seen := 0
	allInt := true
	allFloat := true
	allBool := true
	allDate := true
	allCurrency := true

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
		if !looksLikeCurrency(value) {
			allCurrency = false
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
		if allCurrency {
			return KindCurrency
		}
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

func looksLikeCurrency(value string) bool {
	if !strings.ContainsAny(value, "$€£¥") {
		return false
	}
	_, err := strconv.ParseFloat(normalizeNumber(value), 64)
	return err == nil
}

func looksLikeDate(value string) bool {
	// These layouts cover the simple ISO and month/day formats seen in the
	// existing VisiData sample data fixtures.
	// Go layouts are case-sensitive, so the list includes explicit variants for
	// fixture suffixes `PM`, `pm`, and the short `p`.
	for _, layout := range dateLayouts {
		if _, err := time.Parse(layout, value); err == nil {
			return true
		}
	}

	return false
}

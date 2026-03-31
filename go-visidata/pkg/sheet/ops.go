package sheet

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (s *Sheet) DeleteSelectedRowsOrCurrent() int {
	rows := s.selectedRows()
	if len(rows) == 0 {
		if s.CursorRow < 0 || s.CursorRow >= len(s.Rows) {
			return 0
		}
		rows = []int{s.CursorRow}
	}
	return s.DeleteRows(rows)
}

func (s *Sheet) DeleteRows(rows []int) int {
	if len(rows) == 0 || len(s.Rows) == 0 {
		return 0
	}

	seen := make(map[int]struct{}, len(rows))
	filtered := make([]int, 0, len(rows))
	for _, row := range rows {
		if row < 0 || row >= len(s.Rows) {
			continue
		}
		if _, ok := seen[row]; ok {
			continue
		}
		seen[row] = struct{}{}
		filtered = append(filtered, row)
	}
	if len(filtered) == 0 {
		return 0
	}
	sort.Ints(filtered)
	s.pushUndo("delete rows")
	s.copyRows(filtered, "deleted rows")

	deleteSet := make(map[int]struct{}, len(filtered))
	for _, row := range filtered {
		deleteSet[row] = struct{}{}
	}

	newRows := make([]Row, 0, len(s.Rows)-len(filtered))
	newSelected := make([]bool, 0, len(s.Selected)-len(filtered))
	newRowIDs := make([]int, 0, len(s.rowIDs)-len(filtered))
	for i := range s.Rows {
		if _, ok := deleteSet[i]; ok {
			continue
		}
		newRows = append(newRows, s.Rows[i])
		newSelected = append(newSelected, false)
		newRowIDs = append(newRowIDs, s.rowIDs[i])
	}

	s.Rows = newRows
	s.Selected = newSelected
	s.rowIDs = newRowIDs
	s.refreshSearch()
	s.clampCursor()
	return len(filtered)
}

func (s *Sheet) ToggleSort(columnIndex int, direction SortDirection) {
	if columnIndex < 0 || columnIndex >= len(s.Columns) {
		return
	}
	if s.SortState.ColumnIndex == columnIndex && s.SortState.Direction == direction {
		s.ClearSort()
		return
	}
	s.pushUndo("sort rows")

	type sortableRow struct {
		row      Row
		selected bool
		rowID    int
	}

	rows := make([]sortableRow, len(s.Rows))
	for i := range s.Rows {
		rows[i] = sortableRow{
			row:      s.Rows[i],
			selected: s.IsSelected(i),
			rowID:    s.rowIDs[i],
		}
	}

	sort.SliceStable(rows, func(i, j int) bool {
		left := cellAt(rows[i].row, columnIndex)
		right := cellAt(rows[j].row, columnIndex)
		cmp := compareValues(s.Columns[columnIndex].EffectiveKind(), left, right)
		if cmp == 0 {
			return rows[i].rowID < rows[j].rowID
		}
		if direction == SortDesc {
			return cmp > 0
		}
		return cmp < 0
	})

	for i, row := range rows {
		s.Rows[i] = row.row
		s.Selected[i] = row.selected
		s.rowIDs[i] = row.rowID
	}

	s.SortState = SortState{ColumnIndex: columnIndex, Direction: direction}
	s.refreshSearch()
	s.clampCursor()
}

func (s *Sheet) ClearSort() {
	if len(s.Rows) == 0 {
		s.SortState = SortState{}
		return
	}
	if s.SortState.Direction == SortNone {
		return
	}
	s.pushUndo("clear sort")

	type sortableRow struct {
		row      Row
		selected bool
		rowID    int
	}

	rows := make([]sortableRow, len(s.Rows))
	for i := range s.Rows {
		rows[i] = sortableRow{
			row:      s.Rows[i],
			selected: s.IsSelected(i),
			rowID:    s.rowIDs[i],
		}
	}

	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].rowID < rows[j].rowID
	})

	for i, row := range rows {
		s.Rows[i] = row.row
		s.Selected[i] = row.selected
		s.rowIDs[i] = row.rowID
	}

	s.SortState = SortState{}
	s.refreshSearch()
	s.clampCursor()
}

func (s *Sheet) Search(query string) int {
	query = strings.TrimSpace(query)
	if query == "" {
		s.ClearSearch()
		return 0
	}

	lowerQuery := strings.ToLower(query)
	matches := make([]Position, 0)
	for rowIndex := range s.Rows {
		for _, colIndex := range s.VisibleColumnIndices() {
			if strings.Contains(strings.ToLower(s.Cell(rowIndex, colIndex)), lowerQuery) {
				matches = append(matches, Position{Row: rowIndex, Col: colIndex})
			}
		}
	}

	s.SearchState = SearchState{
		Query:        query,
		Matches:      matches,
		CurrentMatch: 0,
	}
	if len(matches) > 0 {
		s.CursorRow = matches[0].Row
		s.CursorCol = matches[0].Col
		s.clampCursor()
	}
	return len(matches)
}

func (s *Sheet) ClearSearch() {
	s.SearchState = SearchState{}
}

func (s *Sheet) NextMatch(delta int) bool {
	if len(s.SearchState.Matches) == 0 {
		return false
	}

	count := len(s.SearchState.Matches)
	s.SearchState.CurrentMatch = ((s.SearchState.CurrentMatch+delta)%count + count) % count
	pos := s.SearchState.Matches[s.SearchState.CurrentMatch]
	s.CursorRow = pos.Row
	s.CursorCol = pos.Col
	s.clampCursor()
	return true
}

func (s *Sheet) IsSearchMatch(row, col int) bool {
	for _, match := range s.SearchState.Matches {
		if match.Row == row && match.Col == col {
			return true
		}
	}
	return false
}

func (s *Sheet) refreshSearch() {
	if s.SearchState.Query != "" {
		s.Search(s.SearchState.Query)
	}
}

func cellAt(row Row, columnIndex int) string {
	if columnIndex < 0 || columnIndex >= len(row) {
		return ""
	}
	return row[columnIndex]
}

func compareValues(kind ValueKind, left, right string) int {
	left = strings.TrimSpace(left)
	right = strings.TrimSpace(right)

	switch {
	case left == "" && right == "":
		return 0
	case left == "":
		return -1
	case right == "":
		return 1
	}

	switch kind {
	case KindInt:
		return compareInts(left, right)
	case KindFloat, KindCurrency:
		return compareFloats(left, right)
	case KindBool:
		return compareBools(left, right)
	case KindDate:
		return compareDates(left, right)
	default:
		return strings.Compare(strings.ToLower(left), strings.ToLower(right))
	}
}

func compareInts(left, right string) int {
	lv, lerr := strconv.ParseInt(normalizeNumber(left), 10, 64)
	rv, rerr := strconv.ParseInt(normalizeNumber(right), 10, 64)
	if lerr == nil && rerr == nil {
		switch {
		case lv < rv:
			return -1
		case lv > rv:
			return 1
		default:
			return 0
		}
	}
	return strings.Compare(strings.ToLower(left), strings.ToLower(right))
}

func compareFloats(left, right string) int {
	lv, lerr := strconv.ParseFloat(normalizeNumber(left), 64)
	rv, rerr := strconv.ParseFloat(normalizeNumber(right), 64)
	if lerr == nil && rerr == nil {
		switch {
		case lv < rv:
			return -1
		case lv > rv:
			return 1
		default:
			return 0
		}
	}
	return strings.Compare(strings.ToLower(left), strings.ToLower(right))
}

func compareBools(left, right string) int {
	lv, lerr := strconv.ParseBool(strings.ToLower(left))
	rv, rerr := strconv.ParseBool(strings.ToLower(right))
	if lerr == nil && rerr == nil {
		switch {
		case !lv && rv:
			return -1
		case lv && !rv:
			return 1
		default:
			return 0
		}
	}
	return strings.Compare(strings.ToLower(left), strings.ToLower(right))
}

func compareDates(left, right string) int {
	lv, lerr := parseDate(left)
	rv, rerr := parseDate(right)
	if lerr == nil && rerr == nil {
		switch {
		case lv.Before(rv):
			return -1
		case lv.After(rv):
			return 1
		default:
			return 0
		}
	}
	return strings.Compare(strings.ToLower(left), strings.ToLower(right))
}

func parseDate(value string) (time.Time, error) {
	for _, layout := range dateLayouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported date format: %q", value)
}

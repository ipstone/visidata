package sheet

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func (s *Sheet) OverrideColumnKind(columnIndex int, kind ValueKind) error {
	if columnIndex < 0 || columnIndex >= len(s.Columns) {
		return fmt.Errorf("column %d out of range", columnIndex)
	}
	s.Columns[columnIndex].OverrideKind = kind
	s.refreshSearch()
	return nil
}

func (s *Sheet) RenameColumn(columnIndex int, name string) error {
	if columnIndex < 0 || columnIndex >= len(s.Columns) {
		return fmt.Errorf("column %d out of range", columnIndex)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("column name cannot be empty")
	}
	s.Columns[columnIndex].Name = name
	return nil
}

func (s *Sheet) SetColumnWidth(columnIndex int, width int) error {
	if columnIndex < 0 || columnIndex >= len(s.Columns) {
		return fmt.Errorf("column %d out of range", columnIndex)
	}
	if width < 1 {
		return fmt.Errorf("width must be at least 1")
	}
	s.Columns[columnIndex].Width = width
	return nil
}

func (s *Sheet) ToggleHidden(columnIndex int) error {
	if columnIndex < 0 || columnIndex >= len(s.Columns) {
		return fmt.Errorf("column %d out of range", columnIndex)
	}
	if !s.Columns[columnIndex].Hidden && len(s.VisibleColumnIndices()) <= 1 {
		return fmt.Errorf("cannot hide the last visible column")
	}
	s.Columns[columnIndex].Hidden = !s.Columns[columnIndex].Hidden
	s.refreshSearch()
	s.clampCursor()
	return nil
}

func (s *Sheet) ShowAllColumns() int {
	count := 0
	for i := range s.Columns {
		if s.Columns[i].Hidden {
			s.Columns[i].Hidden = false
			count++
		}
	}
	s.refreshSearch()
	s.clampCursor()
	return count
}

func (s *Sheet) SelectByRegex(columnIndex int, pattern string, invert bool) (int, error) {
	if columnIndex < 0 || columnIndex >= len(s.Columns) {
		return 0, fmt.Errorf("column %d out of range", columnIndex)
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return 0, err
	}
	count := 0
	for rowIndex := range s.Rows {
		matched := re.MatchString(s.Cell(rowIndex, columnIndex))
		if invert {
			matched = !matched
		}
		if matched && !s.Selected[rowIndex] {
			s.Selected[rowIndex] = true
			count++
		}
	}
	return count, nil
}

func (s *Sheet) UnselectByRegex(columnIndex int, pattern string, invert bool) (int, error) {
	if columnIndex < 0 || columnIndex >= len(s.Columns) {
		return 0, fmt.Errorf("column %d out of range", columnIndex)
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return 0, err
	}
	count := 0
	for rowIndex := range s.Rows {
		matched := re.MatchString(s.Cell(rowIndex, columnIndex))
		if invert {
			matched = !matched
		}
		if matched && s.Selected[rowIndex] {
			s.Selected[rowIndex] = false
			count++
		}
	}
	return count, nil
}

func visibleRowValues(row Row, cols []int) []string {
	values := make([]string, 0, len(cols))
	for _, col := range cols {
		values = append(values, cellAt(row, col))
	}
	return values
}

func ParseWidth(value string) (int, error) {
	return strconv.Atoi(strings.TrimSpace(value))
}

package sheet

import (
	"fmt"
	"regexp"
)

func (s *Sheet) AddRegexCaptureColumns(columnIndex int, pattern string) ([]string, error) {
	if columnIndex < 0 || columnIndex >= len(s.Columns) {
		return nil, fmt.Errorf("column %d out of range", columnIndex)
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	count := re.NumSubexp()
	if count == 0 {
		return nil, fmt.Errorf("regex capture needs at least one capture group")
	}

	names := make([]string, count)
	subexpNames := re.SubexpNames()
	base := s.Columns[columnIndex].Name
	for i := 0; i < count; i++ {
		name := ""
		if i+1 < len(subexpNames) {
			name = subexpNames[i+1]
		}
		if name == "" {
			name = fmt.Sprintf("%s_capture%d", base, i+1)
		}
		names[i] = uniqueColumnName(s, name)
	}

	s.pushUndo("regex capture columns")
	for _, name := range names {
		s.Columns = append(s.Columns, Column{Name: name, Kind: KindString})
	}
	for rowIndex := range s.Rows {
		matches := re.FindStringSubmatch(s.Cell(rowIndex, columnIndex))
		for capture := 0; capture < count; capture++ {
			value := ""
			if capture+1 < len(matches) {
				value = matches[capture+1]
			}
			s.Rows[rowIndex] = append(s.Rows[rowIndex], value)
		}
	}
	s.CursorCol = len(s.Columns) - count
	s.refreshSearch()
	s.clampCursor()
	return names, nil
}

func (s *Sheet) AddRegexSplitColumns(columnIndex int, pattern string) ([]string, error) {
	if columnIndex < 0 || columnIndex >= len(s.Columns) {
		return nil, fmt.Errorf("column %d out of range", columnIndex)
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	maxParts := 0
	partsByRow := make([][]string, len(s.Rows))
	for rowIndex := range s.Rows {
		parts := re.Split(s.Cell(rowIndex, columnIndex), -1)
		partsByRow[rowIndex] = parts
		if len(parts) > maxParts {
			maxParts = len(parts)
		}
	}
	if maxParts <= 1 {
		return nil, fmt.Errorf("regex split produced only one part")
	}

	base := s.Columns[columnIndex].Name
	names := make([]string, maxParts)
	for i := 0; i < maxParts; i++ {
		names[i] = uniqueColumnName(s, fmt.Sprintf("%s_split%d", base, i+1))
	}

	s.pushUndo("regex split columns")
	for _, name := range names {
		s.Columns = append(s.Columns, Column{Name: name, Kind: KindString})
	}
	for rowIndex := range s.Rows {
		parts := partsByRow[rowIndex]
		for i := 0; i < maxParts; i++ {
			value := ""
			if i < len(parts) {
				value = parts[i]
			}
			s.Rows[rowIndex] = append(s.Rows[rowIndex], value)
		}
	}
	s.CursorCol = len(s.Columns) - maxParts
	s.refreshSearch()
	s.clampCursor()
	return names, nil
}

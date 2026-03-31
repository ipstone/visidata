package sheet

import "fmt"

func (s *Sheet) SetCell(rowIndex, columnIndex int, value string) error {
	if rowIndex < 0 || rowIndex >= len(s.Rows) {
		return fmt.Errorf("row %d out of range", rowIndex)
	}
	if columnIndex < 0 || columnIndex >= len(s.Columns) {
		return fmt.Errorf("column %d out of range", columnIndex)
	}
	if cellAt(s.Rows[rowIndex], columnIndex) == value {
		return nil
	}
	s.pushUndo("edit cell")
	s.Rows[rowIndex][columnIndex] = value
	s.refreshSearch()
	s.clampCursor()
	return nil
}

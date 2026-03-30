package sheet

import "strings"

func (s *Sheet) CopyCell(row, col int) bool {
	if row < 0 || row >= len(s.Rows) || col < 0 || col >= len(s.Columns) {
		return false
	}
	value := s.Cell(row, col)

	s.Clipboard = Clipboard{
		Content: value,
		Kind:    "cell",
		Count:   1,
	}
	return true
}

func (s *Sheet) CopySelectedRowsOrCurrent() bool {
	rows := s.selectedRows()
	kind := "selected rows"
	if len(rows) == 0 {
		if s.CursorRow < 0 || s.CursorRow >= len(s.Rows) {
			return false
		}
		rows = []int{s.CursorRow}
		kind = "row"
	}

	content := make([]string, 0, len(rows))
	cols := s.VisibleColumnIndices()
	for _, rowIndex := range rows {
		content = append(content, strings.Join(visibleRowValues(s.Rows[rowIndex], cols), "\t"))
	}

	s.Clipboard = Clipboard{
		Content: strings.Join(content, "\n"),
		Kind:    kind,
		Count:   len(rows),
	}
	return true
}

func (s *Sheet) copyRows(rows []int, kind string) bool {
	if len(rows) == 0 {
		return false
	}

	content := make([]string, 0, len(rows))
	cols := s.VisibleColumnIndices()
	for _, rowIndex := range rows {
		if rowIndex < 0 || rowIndex >= len(s.Rows) {
			continue
		}
		content = append(content, strings.Join(visibleRowValues(s.Rows[rowIndex], cols), "\t"))
	}
	if len(content) == 0 {
		return false
	}

	s.Clipboard = Clipboard{
		Content: strings.Join(content, "\n"),
		Kind:    kind,
		Count:   len(content),
	}
	return true
}

func (s *Sheet) ClipboardPreview(limit int) string {
	if s.Clipboard.Content == "" {
		return ""
	}
	preview := strings.ReplaceAll(s.Clipboard.Content, "\t", " ⇥ ")
	preview = strings.ReplaceAll(preview, "\n", " ⏎ ")
	runes := []rune(preview)
	if limit > 0 && len(runes) > limit {
		return string(runes[:limit]) + "…"
	}
	return preview
}

func (s *Sheet) selectedRows() []int {
	rows := make([]int, 0, s.SelectedCount())
	for i, selected := range s.Selected {
		if selected {
			rows = append(rows, i)
		}
	}
	return rows
}

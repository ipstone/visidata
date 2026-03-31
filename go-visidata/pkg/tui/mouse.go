package tui

func (a *App) handleMousePrimary(x, y int) {
	width, _ := a.size()
	if y < 0 {
		return
	}
	sidebarWidth := a.sidebarWidth(width)
	if sidebarWidth > 0 && x < sidebarWidth-1 {
		a.handleSidebarClick(y)
		return
	}
	contentX, contentWidth := a.contentFrame(width)

	if y == 1 || y == 2 {
		if col, ok := a.columnAtX(contentWidth, x-contentX); ok {
			a.Sheet.SetCursorCol(col)
		}
		return
	}

	rowIndex := a.rowAtY(y)
	if rowIndex < 0 || rowIndex >= len(a.Sheet.Rows) {
		return
	}

	if col, ok := a.columnAtX(contentWidth, x-contentX); ok {
		a.Sheet.SetCursorCol(col)
	}
	a.Sheet.SetCursorRow(rowIndex)

	if !a.dragSelecting {
		a.dragSelecting = true
		a.dragAnchorRow = rowIndex
		return
	}

	a.selectDragRange(a.dragAnchorRow, rowIndex)
}

func (a *App) handleSidebarClick(y int) {
	row := y - 2
	if row < 0 || row >= len(a.stack) {
		return
	}
	a.sidebarIndex = row
	a.sidebarVisible = true
	a.activateSidebarSelection()
	a.closeSidebar()
}

func (a *App) selectDragRange(start, end int) {
	if len(a.Sheet.Rows) == 0 {
		return
	}
	if start > end {
		start, end = end, start
	}
	if start < 0 {
		start = 0
	}
	if end >= len(a.Sheet.Rows) {
		end = len(a.Sheet.Rows) - 1
	}
	a.Sheet.ClearSelection()
	for row := start; row <= end; row++ {
		a.Sheet.ToggleSelected(row)
	}
}

func (a *App) rowAtY(y int) int {
	if y < 3 {
		return -1
	}
	return a.RowOffset + (y - 3)
}

func (a *App) columnAtX(width, x int) (int, bool) {
	if x < rowPrefixWidth {
		return 0, false
	}

	visibleCols := a.visibleColumns(width)
	cursor := rowPrefixWidth
	for i, col := range visibleCols {
		if i > 0 {
			cursor += 3
		}
		if x >= cursor && x < cursor+a.colWidths[col] {
			return col, true
		}
		cursor += a.colWidths[col]
	}
	return 0, false
}

package tui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/ipstone/visidata/go-visidata/pkg/sheet"
	"github.com/mattn/go-runewidth"
)

const rowPrefixWidth = 2

func (a *App) Draw() {
	if a.Screen == nil {
		return
	}

	width, height := a.size()
	a.ensureVisible(width, height)
	a.Screen.Clear()

	headerStyle := tcell.StyleDefault.Bold(true)
	statusStyle := tcell.StyleDefault.Reverse(true)

	a.drawText(0, 0, width, a.Sheet.Summary(), headerStyle)
	if height < 2 {
		a.Screen.Show()
		return
	}

	visibleCols := a.visibleColumns(width)
	if len(visibleCols) == 0 {
		a.drawText(0, height-1, width, a.statusLine(), statusStyle)
		a.Screen.Show()
		return
	}

	a.drawRow(1, width, visibleCols, func(col int) string {
		return fmt.Sprintf("%s <%s>", a.Sheet.Columns[col].Name, a.Sheet.Columns[col].Kind)
	}, headerStyle, -1)

	if height > 2 {
		a.drawRow(2, width, visibleCols, func(col int) string {
			return strings.Repeat("-", a.colWidths[col])
		}, tcell.StyleDefault, -1)
	}

	dataHeight := max(0, height-4)
	for i := 0; i < dataHeight; i++ {
		rowIndex := a.RowOffset + i
		y := 3 + i
		if rowIndex >= len(a.Sheet.Rows) {
			break
		}

		a.drawRow(y, width, visibleCols, func(col int) string {
			return a.Sheet.Cell(rowIndex, col)
		}, tcell.StyleDefault, rowIndex)
	}

	if height > 0 {
		a.drawText(0, height-1, width, a.statusLine(), statusStyle)
	}

	a.Screen.Show()
}

func (a *App) drawRow(y, width int, cols []int, valueFn func(col int) string, baseStyle tcell.Style, cursorRow int) {
	prefix := "  "
	if cursorRow >= 0 && a.Sheet.IsSelected(cursorRow) {
		prefix = "* "
	}

	x := a.drawText(0, y, rowPrefixWidth, prefix, baseStyle)
	for i, col := range cols {
		if i > 0 {
			x = a.drawText(x, y, width-x, " | ", baseStyle)
		}
		style := baseStyle
		if cursorRow >= 0 && a.Sheet.IsSelected(cursorRow) {
			style = style.Foreground(tcell.ColorLightGreen)
		}
		if cursorRow >= 0 && a.Sheet.IsSearchMatch(cursorRow, col) {
			style = style.Background(tcell.ColorDarkCyan)
		}
		if cursorRow == a.Sheet.CursorRow && col == a.Sheet.CursorCol {
			style = style.Reverse(true).Bold(true)
		}
		x = a.drawText(x, y, width-x, padRight(valueFn(col), a.colWidths[col]), style)
		if x >= width {
			return
		}
	}
}

func (a *App) drawText(x, y, width int, text string, style tcell.Style) int {
	if width <= 0 {
		return x
	}

	remaining := width
	for _, r := range text {
		w := runewidth.RuneWidth(r)
		if w == 0 {
			w = 1
		}
		if w > remaining {
			break
		}
		a.Screen.SetContent(x, y, r, nil, style)
		x += w
		remaining -= w
	}
	for remaining > 0 {
		a.Screen.SetContent(x, y, ' ', nil, style)
		x++
		remaining--
	}
	return x
}

func (a *App) statusLine() string {
	if a.searchMode {
		return fmt.Sprintf("Search: %s", string(a.searchQuery))
	}

	parts := []string{
		a.Sheet.Name,
		fmt.Sprintf("row %d/%d", min(a.Sheet.CursorRow+1, len(a.Sheet.Rows)), len(a.Sheet.Rows)),
		fmt.Sprintf("col %d/%d", min(a.Sheet.CursorCol+1, len(a.Sheet.Columns)), len(a.Sheet.Columns)),
		fmt.Sprintf("sel %d", a.Sheet.SelectedCount()),
	}
	if a.Sheet.SortState.Direction != sheet.SortNone && a.Sheet.SortState.ColumnIndex < len(a.Sheet.Columns) {
		dir := "↑"
		if a.Sheet.SortState.Direction == sheet.SortDesc {
			dir = "↓"
		}
		parts = append(parts, fmt.Sprintf("sort %s%s", a.Sheet.Columns[a.Sheet.SortState.ColumnIndex].Name, dir))
	}
	if query := a.Sheet.SearchState.Query; query != "" {
		parts = append(parts, fmt.Sprintf("search %q %d/%d", query, min(a.Sheet.SearchState.CurrentMatch+1, len(a.Sheet.SearchState.Matches)), len(a.Sheet.SearchState.Matches)))
	}
	parts = append(parts, "q quit", "/ search", "[ ] sort", "s/t/u select")
	return strings.Join(parts, "  ")
}

func (a *App) visibleColumns(width int) []int {
	if width <= 0 {
		return nil
	}

	var cols []int
	used := 0
	for col := a.ColOffset; col < len(a.Sheet.Columns); col++ {
		next := a.colWidths[col]
		if len(cols) > 0 {
			next += 3
		}
		if len(cols) > 0 && used+next > max(0, width-rowPrefixWidth) {
			break
		}
		used += next
		cols = append(cols, col)
	}

	if len(cols) == 0 && a.ColOffset < len(a.Sheet.Columns) {
		return []int{a.ColOffset}
	}
	return cols
}

func (a *App) ensureVisible(width, height int) {
	if a.Sheet.CursorRow < a.RowOffset {
		a.RowOffset = a.Sheet.CursorRow
	}

	pageRows := max(1, height-4)
	if a.Sheet.CursorRow >= a.RowOffset+pageRows {
		a.RowOffset = a.Sheet.CursorRow - pageRows + 1
	}
	if a.RowOffset < 0 {
		a.RowOffset = 0
	}

	if a.Sheet.CursorCol < a.ColOffset {
		a.ColOffset = a.Sheet.CursorCol
	}

	for {
		cols := a.visibleColumns(width)
		if len(cols) == 0 || a.Sheet.CursorCol <= cols[len(cols)-1] {
			break
		}
		a.ColOffset++
	}
}

func columnWidths(sh *sheet.Sheet) []int {
	widths := make([]int, len(sh.Columns))
	for i, col := range sh.Columns {
		widths[i] = displayWidth(fmt.Sprintf("%s <%s>", col.Name, col.Kind))
	}

	for _, row := range sh.Rows {
		for i := range sh.Columns {
			if i < len(row) && displayWidth(row[i]) > widths[i] {
				widths[i] = displayWidth(row[i])
			}
		}
	}
	return widths
}

func padRight(value string, width int) string {
	if displayWidth(value) >= width {
		return value
	}
	return value + strings.Repeat(" ", width-displayWidth(value))
}

func displayWidth(value string) int {
	return runewidth.StringWidth(value)
}

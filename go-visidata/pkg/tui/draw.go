package tui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/ipstone/visidata/go-visidata/pkg/sheet"
	"github.com/mattn/go-runewidth"
)

const rowPrefixWidth = 2
const clipboardPreviewMaxLen = 24

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
		return a.Sheet.Columns[col].Label()
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
	if a.mode != inputModeNone {
		return fmt.Sprintf("%s: %s", a.inputPrompt(), string(a.inputValue))
	}

	parts := []string{
		a.Sheet.Name,
		fmt.Sprintf("row %d/%d", min(a.Sheet.CursorRow+1, len(a.Sheet.Rows)), len(a.Sheet.Rows)),
		fmt.Sprintf("col %d/%d", min(visibleColumnOrdinal(a.Sheet, a.Sheet.CursorCol), len(a.Sheet.VisibleColumnIndices())), len(a.Sheet.VisibleColumnIndices())),
		fmt.Sprintf("sel %d", a.Sheet.SelectedCount()),
	}
	if hidden := a.Sheet.HiddenColumnCount(); hidden > 0 {
		parts = append(parts, fmt.Sprintf("hidden %d", hidden))
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
	if clip := a.Sheet.ClipboardPreview(clipboardPreviewMaxLen); clip != "" {
		label := "copy"
		switch a.Sheet.Clipboard.Kind {
		case "cell":
			label = "cell"
		case "row", "selected rows", "deleted rows":
			label = fmt.Sprintf("%s %d", a.Sheet.Clipboard.Kind, a.Sheet.Clipboard.Count)
		}
		parts = append(parts, fmt.Sprintf("%s %q", label, clip))
	}
	if a.Sheet.Status != "" {
		parts = append(parts, a.Sheet.Status)
	}
	parts = append(parts, controlHints()...)
	return strings.Join(parts, "  ")
}

func (a *App) visibleColumns(width int) []int {
	if width <= 0 {
		return nil
	}

	var cols []int
	used := 0
	for col := a.ColOffset; col < len(a.Sheet.Columns); col++ {
		if a.Sheet.Columns[col].Hidden {
			continue
		}
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

	if len(cols) == 0 {
		for col := a.ColOffset; col < len(a.Sheet.Columns); col++ {
			if !a.Sheet.Columns[col].Hidden {
				return []int{col}
			}
		}
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
		widths[i] = displayWidth(col.Label())
		if col.Width > 0 && col.Width > widths[i] {
			widths[i] = col.Width
		}
	}

	for _, row := range sh.Rows {
		for i := range sh.Columns {
			if i < len(row) && displayWidth(row[i]) > widths[i] {
				widths[i] = displayWidth(row[i])
			}
			if sh.Columns[i].Width > 0 {
				widths[i] = sh.Columns[i].Width
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

func visibleColumnOrdinal(sh *sheet.Sheet, col int) int {
	ordinal := 0
	for i, column := range sh.Columns {
		if column.Hidden {
			continue
		}
		ordinal++
		if i == col {
			return ordinal
		}
	}
	if ordinal == 0 {
		return 0
	}
	return ordinal
}

func controlHints() []string {
	return []string{
		"q quit",
		"/ search",
		"[ ] sort",
		"s/t/u select",
		"c/C copy",
		"d delete",
		"S save",
		"- hide",
		"H show-all",
		"^ rename",
		"_ width",
		"~ # % $ @ type",
		"| / \\ regex",
	}
}

func (a *App) inputPrompt() string {
	switch a.mode {
	case inputModeSearch:
		return "Search"
	case inputModeRename:
		return "Rename column"
	case inputModeResize:
		return "Column width"
	case inputModeRegexSelect:
		return "Select regex"
	case inputModeRegexUnselect:
		return "Unselect regex"
	default:
		return "Input"
	}
}

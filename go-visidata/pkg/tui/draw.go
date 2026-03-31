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
	contentX, contentWidth := a.contentFrame(width)

	headerStyle := tcell.StyleDefault.Foreground(a.theme.HeaderFG).Background(a.theme.HeaderBG).Bold(true)
	statusStyle := tcell.StyleDefault.Foreground(a.theme.StatusFG).Background(a.theme.StatusBG)
	bodyStyle := tcell.StyleDefault.Foreground(a.theme.BodyFG).Background(a.theme.BodyBG)

	a.drawText(0, 0, width, a.Sheet.Summary(), headerStyle)
	if height < 2 {
		a.Screen.Show()
		return
	}
	if a.sidebarVisible {
		a.drawSidebar(width, height)
	}

	if a.Sheet.Graph != nil {
		a.drawGraph(contentX, contentWidth, height, bodyStyle, headerStyle)
		if height > 0 {
			a.drawText(0, height-1, width, a.statusLine(), statusStyle)
		}
		if a.mode == inputModeMenu {
			a.drawMenu(width, height)
		}
		if a.mode == inputModeCommandPalette {
			a.drawCommandPalette(width, height)
		}
		a.Screen.Show()
		return
	}

	visibleCols := a.visibleColumns(contentWidth)
	if len(visibleCols) == 0 {
		a.drawText(0, height-1, width, a.statusLine(), statusStyle)
		a.Screen.Show()
		return
	}

	a.drawRow(contentX, 1, contentWidth, visibleCols, func(col int) string {
		return a.Sheet.Columns[col].Label()
	}, headerStyle, -1)

	if height > 2 {
		a.drawRow(contentX, 2, contentWidth, visibleCols, func(col int) string {
			return strings.Repeat("-", a.colWidths[col])
		}, bodyStyle, -1)
	}

	dataHeight := max(0, height-4)
	for i := 0; i < dataHeight; i++ {
		rowIndex := a.RowOffset + i
		y := 3 + i
		if rowIndex >= len(a.Sheet.Rows) {
			break
		}

		a.drawRow(contentX, y, contentWidth, visibleCols, func(col int) string {
			return a.Sheet.Cell(rowIndex, col)
		}, bodyStyle, rowIndex)
	}

	if height > 0 {
		a.drawText(0, height-1, width, a.statusLine(), statusStyle)
	}
	if a.mode == inputModeMenu {
		a.drawMenu(width, height)
	}
	if a.mode == inputModeCommandPalette {
		a.drawCommandPalette(width, height)
	}

	a.Screen.Show()
}

func (a *App) drawMenu(width, height int) {
	if len(menuCatalog) == 0 || height < 4 {
		return
	}

	barStyle := tcell.StyleDefault.Foreground(a.theme.OverlayFG).Background(a.theme.OverlayBG)
	selectedStyle := barStyle.Bold(true)
	x := 0
	for i, group := range menuCatalog {
		style := barStyle
		if i == a.menuIndex {
			style = selectedStyle
		}
		label := " " + group.Label + " "
		x = a.drawText(x, 1, min(displayWidth(label), width-x), label, style)
		if x >= width {
			break
		}
	}

	group := menuCatalog[a.menuIndex]
	boxWidth := 0
	for _, item := range group.Items {
		if w := displayWidth(item.Label) + 4; w > boxWidth {
			boxWidth = w
		}
	}
	boxWidth = max(boxWidth, displayWidth(group.Label)+4)
	boxWidth = min(boxWidth, width)
	x0 := 0
	for i := 0; i < a.menuIndex; i++ {
		x0 += displayWidth(" " + menuCatalog[i].Label + " ")
	}
	y0 := 2
	for i, item := range group.Items {
		if y0+i >= height-1 {
			break
		}
		style := barStyle
		prefix := "  "
		if i == a.menuItemIndex {
			style = selectedStyle
			prefix = "> "
		}
		a.drawText(x0, y0+i, min(boxWidth, width-x0), prefix+item.Label, style)
	}
}

func (a *App) drawCommandPalette(width, height int) {
	if height < 4 {
		return
	}

	matches := a.commandPaletteMatches()
	lines := min(5, len(matches))
	if lines == 0 {
		lines = 1
	}
	boxWidth := min(width, max(36, width*2/3))
	if boxWidth <= 0 {
		return
	}
	x0 := max(0, (width-boxWidth)/2)
	y0 := max(1, height-lines-3)
	style := tcell.StyleDefault.Foreground(a.theme.OverlayFG).Background(a.theme.OverlayBG)
	title := " Command Palette: " + string(a.inputValue)
	a.drawText(x0, y0, boxWidth, title, style)

	if len(matches) == 0 {
		a.drawText(x0, y0+1, boxWidth, "no commands matched", style)
		return
	}

	selected := a.paletteSelection(matches)
	for i := 0; i < lines; i++ {
		match := matches[i]
		lineStyle := style
		prefix := "  "
		if i == selected {
			lineStyle = style.Bold(true)
			prefix = "> "
		}
		text := prefix + match.Name + " [" + match.Keys + "] " + match.Help
		a.drawText(x0, y0+1+i, boxWidth, text, lineStyle)
	}
}

func (a *App) drawSidebar(width, height int) {
	sidebarWidth := a.sidebarWidth(width)
	if sidebarWidth == 0 || height < 3 {
		return
	}

	titleStyle := tcell.StyleDefault.Foreground(a.theme.OverlayFG).Background(a.theme.OverlayBG).Bold(true)
	bodyStyle := tcell.StyleDefault.Foreground(a.theme.OverlayFG).Background(a.theme.OverlayBG)
	selectedStyle := bodyStyle.Bold(true)
	currentStyle := bodyStyle.Foreground(a.theme.AccentFG)

	a.drawText(0, 1, sidebarWidth-1, " Sheets ", titleStyle)
	for y := 1; y < height-1; y++ {
		a.drawText(sidebarWidth-1, y, 1, "|", bodyStyle)
	}
	for i, sh := range a.stack {
		y := 2 + i
		if y >= height-1 {
			break
		}
		style := bodyStyle
		prefix := "  "
		if i == a.sidebarIndex {
			style = selectedStyle
			prefix = "> "
		}
		if sh == a.Sheet {
			style = currentStyle
			if i == a.sidebarIndex {
				style = style.Bold(true)
			}
			prefix = "* "
		}
		a.drawText(0, y, sidebarWidth-1, prefix+sh.Name, style)
	}
}

func (a *App) drawRow(x0, y, width int, cols []int, valueFn func(col int) string, baseStyle tcell.Style, cursorRow int) {
	prefix := "  "
	if cursorRow >= 0 && a.Sheet.IsSelected(cursorRow) {
		prefix = "* "
	}

	x := a.drawText(x0, y, rowPrefixWidth, prefix, baseStyle)
	for i, col := range cols {
		if i > 0 {
			x = a.drawText(x, y, min(3, width-x), " | ", baseStyle)
		}
		style := baseStyle
		if cursorRow >= 0 && a.Sheet.IsSelected(cursorRow) {
			style = style.Foreground(a.theme.AccentFG)
		}
		if cursorRow >= 0 && a.Sheet.IsSearchMatch(cursorRow, col) {
			style = style.Background(a.theme.SearchBG)
		}
		if cursorRow == a.Sheet.CursorRow && col == a.Sheet.CursorCol {
			style = style.Foreground(a.theme.CursorFG).Background(a.theme.CursorBG).Bold(true)
		}
		x = a.drawText(x, y, min(a.colWidths[col], width-x), padRight(valueFn(col), a.colWidths[col]), style)
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
	if len(a.stack) > 1 {
		parts = append(parts, fmt.Sprintf("stack %d", len(a.stack)))
	}
	if a.sidebarVisible {
		parts = append(parts, "sidebar")
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
	if a.Sheet.Graph != nil {
		point := min(a.Sheet.CursorRow+1, len(a.Sheet.Rows))
		parts = append(parts,
			fmt.Sprintf("graph %s", a.Sheet.Graph.Kind),
			fmt.Sprintf("x %s", a.Sheet.Graph.XLabel),
			fmt.Sprintf("y %s", a.Sheet.Graph.YLabel),
			fmt.Sprintf("point %d/%d", point, len(a.Sheet.Rows)),
		)
		if a.Sheet.CursorRow >= 0 && a.Sheet.CursorRow < len(a.Sheet.Graph.Points) {
			current := a.Sheet.Graph.Points[a.Sheet.CursorRow]
			parts = append(parts, fmt.Sprintf("%s=(%.2f,%.2f)", current.Label, current.X, current.Y))
		}
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
	_, width = a.contentFrame(width)

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
	ordinal := sheetVisibleColumnOrdinal(sh, col)
	if ordinal == 0 && len(sh.VisibleColumnIndices()) == 0 {
		return 0
	}
	return ordinal
}

func sheetVisibleColumnOrdinal(sh *sheet.Sheet, col int) int {
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
	return ordinal
}

func (a *App) inputPrompt() string {
	switch a.mode {
	case inputModeSearch:
		return "Search"
	case inputModeCommandPalette:
		return "Command"
	case inputModeMenu:
		return "Menu"
	case inputModeEditCell:
		return "Edit cell"
	case inputModeExpr:
		return "Expr column"
	case inputModeRename:
		return "Rename column"
	case inputModeResize:
		return "Column width"
	case inputModeRegexCapture:
		return "Capture regex"
	case inputModeRegexSplit:
		return "Split regex"
	case inputModeRegexSelect:
		return "Select regex"
	case inputModeRegexUnselect:
		return "Unselect regex"
	default:
		return "Input"
	}
}

package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/ipstone/visidata/go-visidata/pkg/sheet"
)

type inputMode string

const (
	inputModeNone          inputMode = ""
	inputModeSearch        inputMode = "search"
	inputModeEditCell      inputMode = "edit-cell"
	inputModeExpr          inputMode = "expr"
	inputModeRename        inputMode = "rename"
	inputModeResize        inputMode = "resize"
	inputModeRegexSelect   inputMode = "regex-select"
	inputModeRegexUnselect inputMode = "regex-unselect"
)

type App struct {
	Screen     tcell.Screen
	Sheet      *sheet.Sheet
	RowOffset  int
	ColOffset  int
	stack      []*sheet.Sheet
	commandLog []commandLogEntry
	mode       inputMode
	inputValue []rune
	colWidths  []int
}

type commandLogEntry struct {
	Index  int
	Keys   string
	Name   string
	Sheet  string
	Detail string
}

func New(sh *sheet.Sheet, screen tcell.Screen) *App {
	return &App{
		Screen:    screen,
		Sheet:     sh,
		stack:     []*sheet.Sheet{sh},
		colWidths: columnWidths(sh),
	}
}

func Run(sh *sheet.Sheet) error {
	screen, err := tcell.NewScreen()
	if err != nil {
		return fmt.Errorf("create screen: %w", err)
	}

	app := New(sh, screen)
	return app.Run()
}

func (a *App) Run() error {
	if err := a.Screen.Init(); err != nil {
		return fmt.Errorf("init screen: %w", err)
	}
	defer a.Screen.Fini()

	a.Screen.Clear()
	for {
		a.Draw()
		ev := a.Screen.PollEvent()
		switch event := ev.(type) {
		case *tcell.EventResize:
			a.Screen.Sync()
		case *tcell.EventKey:
			shouldQuit := a.HandleKey(event)
			if shouldQuit {
				return nil
			}
		}
	}
}

func (a *App) HandleKey(ev *tcell.EventKey) bool {
	if a.mode != inputModeNone {
		return a.handleInputKey(ev)
	}

	pageSize := a.pageSize()

	switch ev.Key() {
	case tcell.KeyCtrlC, tcell.KeyEscape:
		return true
	case tcell.KeyCtrlS:
		a.saveSuggested()
	case tcell.KeyUp:
		a.Sheet.MoveCursorRow(-1)
	case tcell.KeyDown:
		a.Sheet.MoveCursorRow(1)
	case tcell.KeyLeft:
		a.Sheet.MoveCursorCol(-1)
	case tcell.KeyRight:
		a.Sheet.MoveCursorCol(1)
	case tcell.KeyPgUp:
		a.Sheet.MoveCursorRow(-pageSize)
	case tcell.KeyPgDn:
		a.Sheet.MoveCursorRow(pageSize)
	case tcell.KeyHome:
		a.Sheet.SetCursorRow(0)
	case tcell.KeyEnd:
		a.Sheet.SetCursorRow(len(a.Sheet.Rows) - 1)
	case tcell.KeyEnter:
		a.activateCurrentRow()
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'q':
			if a.popSheet() {
				break
			}
			return true
		case '/':
			a.beginInput(inputModeSearch, a.Sheet.SearchState.Query)
		case 'h':
			a.Sheet.MoveCursorCol(-1)
		case 'j':
			a.Sheet.MoveCursorRow(1)
		case 'k':
			a.Sheet.MoveCursorRow(-1)
		case 'l':
			a.Sheet.MoveCursorCol(1)
		case 'g':
			a.Sheet.SetCursorRow(0)
		case 'G':
			a.Sheet.SetCursorRow(len(a.Sheet.Rows) - 1)
		case '[':
			a.Sheet.ToggleSort(a.Sheet.CursorCol, sheet.SortAsc)
		case ']':
			a.Sheet.ToggleSort(a.Sheet.CursorCol, sheet.SortDesc)
		case '=':
			a.beginInput(inputModeExpr, "")
		case '&':
			a.openJoinSheet()
			a.recordCommand("&", "join", a.Sheet.Status)
		case 'f':
			a.pushSheet(a.Sheet.FreezeSheet())
			a.Sheet.Status = fmt.Sprintf("freeze %s", a.Sheet.Name)
			a.recordCommand("f", "freeze", a.Sheet.Status)
		case 'D':
			a.openDedupeSheet()
			a.recordCommand("D", "dedupe", a.Sheet.Status)
		case 'e':
			a.beginEditCellInput()
		case 'm':
			a.openMeltSheet()
			a.recordCommand("m", "melt", a.Sheet.Status)
		case 'n':
			a.Sheet.NextMatch(1)
		case 'N':
			a.Sheet.NextMatch(-1)
		case 's':
			a.Sheet.ToggleSelected(a.Sheet.CursorRow)
		case 't':
			a.Sheet.SelectAll()
		case 'u':
			a.Sheet.ClearSelection()
		case 'c':
			a.Sheet.CopyCell(a.Sheet.CursorRow, a.Sheet.CursorCol)
			a.Sheet.Status = "copied current cell"
		case 'C':
			a.Sheet.CopySelectedRowsOrCurrent()
			a.Sheet.Status = "copied row data"
		case 'd':
			count := a.Sheet.DeleteSelectedRowsOrCurrent()
			if count > 0 {
				a.colWidths = columnWidths(a.Sheet)
				a.Sheet.Status = fmt.Sprintf("deleted %d row(s)", count)
				a.recordCommand("d", "delete", a.Sheet.Status)
			}
		case 'S':
			a.saveSuggested()
			a.recordCommand("S", "save", a.Sheet.Status)
		case 'F':
			a.openFrequencySheet()
			a.recordCommand("F", "frequency", a.Sheet.Status)
		case 'I':
			a.pushSheet(a.Sheet.DescribeSheet())
			a.Sheet.Status = fmt.Sprintf("describe %s", a.Sheet.Name)
			a.recordCommand("I", "describe", a.Sheet.Status)
		case 'W':
			a.openPivotSheet()
			a.recordCommand("W", "pivot", a.Sheet.Status)
		case 'T':
			a.pushSheet(a.Sheet.TransposeSheet())
			a.Sheet.Status = fmt.Sprintf("transpose %s", a.Sheet.Name)
			a.recordCommand("T", "transpose", a.Sheet.Status)
		case 'V':
			a.pushSheet(a.buildSheetsSheet())
			a.Sheet.Status = fmt.Sprintf("sheets %s", a.Sheet.Name)
			a.recordCommand("V", "sheets", a.Sheet.Status)
		case 'M':
			a.pushSheet(a.Sheet.ColumnsSheet())
			a.Sheet.Status = fmt.Sprintf("columns %s", a.Sheet.Name)
			a.recordCommand("M", "columns", a.Sheet.Status)
		case 'O':
			a.pushSheet(a.Sheet.OptionsSheet())
			a.Sheet.Status = fmt.Sprintf("options %s", a.Sheet.Name)
			a.recordCommand("O", "options", a.Sheet.Status)
		case 'P':
			a.recordCommand("P", "cmdlog", "opened command log")
			a.pushSheet(a.buildCommandLogSheet())
			a.Sheet.Status = fmt.Sprintf("cmdlog %s", a.Sheet.Name)
		case '?':
			a.pushSheet(a.buildCommandsSheet())
			a.Sheet.Status = fmt.Sprintf("commands %s", a.Sheet.Name)
			a.recordCommand("?", "commands", a.Sheet.Status)
		case 'R':
			a.redoCurrentSheet()
		case 'U':
			a.undoCurrentSheet()
		case '~':
			if !a.overrideColumnsMetaType(sheet.KindString) {
				a.overrideColumnType(sheet.KindString)
			}
		case '#':
			if !a.overrideColumnsMetaType(sheet.KindInt) {
				a.overrideColumnType(sheet.KindInt)
			}
		case '%':
			if !a.overrideColumnsMetaType(sheet.KindFloat) {
				a.overrideColumnType(sheet.KindFloat)
			}
		case '$':
			if !a.overrideColumnsMetaType(sheet.KindCurrency) {
				a.overrideColumnType(sheet.KindCurrency)
			}
		case '@':
			if !a.overrideColumnsMetaType(sheet.KindDate) {
				a.overrideColumnType(sheet.KindDate)
			}
		case '-':
			if !a.toggleColumnsMetaHidden() {
				name := a.Sheet.Columns[a.Sheet.CursorCol].Name
				if err := a.Sheet.ToggleHidden(a.Sheet.CursorCol); err != nil {
					a.Sheet.Status = fmt.Sprintf("hide failed: %v", err)
				} else {
					a.colWidths = columnWidths(a.Sheet)
					a.Sheet.Status = fmt.Sprintf("toggled hidden for %s", name)
				}
			}
		case 'H':
			if !a.showAllColumnsMeta() {
				count := a.Sheet.ShowAllColumns()
				a.colWidths = columnWidths(a.Sheet)
				a.Sheet.Status = fmt.Sprintf("revealed %d column(s)", count)
				if count > 0 {
					a.recordCommand("H", "show-all-columns", a.Sheet.Status)
				}
			}
		case '^':
			a.beginRenameInput()
		case '_':
			a.beginResizeInput()
		case '|':
			a.beginInput(inputModeRegexSelect, "")
		case '\\':
			a.beginInput(inputModeRegexUnselect, "")
		}
	}

	a.ensureVisible(a.size())
	return false
}

func (a *App) pushSheet(sh *sheet.Sheet) {
	if sh == nil {
		return
	}
	a.stack = append(a.stack, sh)
	a.Sheet = sh
	a.RowOffset = 0
	a.ColOffset = 0
	a.mode = inputModeNone
	a.inputValue = nil
	a.colWidths = columnWidths(sh)
}

func (a *App) popSheet() bool {
	if len(a.stack) <= 1 {
		return false
	}
	a.stack = a.stack[:len(a.stack)-1]
	a.Sheet = a.stack[len(a.stack)-1]
	a.RowOffset = 0
	a.ColOffset = 0
	a.mode = inputModeNone
	a.inputValue = nil
	a.colWidths = columnWidths(a.Sheet)
	a.Sheet.Status = fmt.Sprintf("returned to %s", a.Sheet.Name)
	return true
}

func (a *App) openFrequencySheet() {
	sh, err := a.Sheet.FrequencySheet(a.Sheet.CursorCol)
	if err != nil {
		a.Sheet.Status = fmt.Sprintf("frequency failed: %v", err)
		return
	}
	a.pushSheet(sh)
	a.Sheet.Status = fmt.Sprintf("frequency %s", a.Sheet.Name)
}

func (a *App) openJoinSheet() {
	if len(a.stack) < 2 {
		a.Sheet.Status = "join needs another sheet on the stack"
		return
	}
	other := a.stack[len(a.stack)-2]
	leftColumnIndex := a.Sheet.CursorCol
	rightColumnIndex := matchingColumnIndex(other, a.Sheet.Columns[leftColumnIndex].Name)
	if rightColumnIndex < 0 {
		rightColumnIndex = other.CursorCol
	}
	if rightColumnIndex < 0 || rightColumnIndex >= len(other.Columns) {
		a.Sheet.Status = "join needs a valid column in the previous sheet"
		return
	}
	sh, err := a.Sheet.JoinSheet(other, leftColumnIndex, rightColumnIndex)
	if err != nil {
		a.Sheet.Status = fmt.Sprintf("join failed: %v", err)
		return
	}
	a.pushSheet(sh)
	a.Sheet.Status = fmt.Sprintf("join %s", a.Sheet.Name)
}

func matchingColumnIndex(sh *sheet.Sheet, name string) int {
	for i, col := range sh.Columns {
		if col.Name == name {
			return i
		}
	}
	return -1
}

func (a *App) openPivotSheet() {
	sh, err := a.Sheet.PivotSheet(a.Sheet.CursorCol)
	if err != nil {
		a.Sheet.Status = fmt.Sprintf("pivot failed: %v", err)
		return
	}
	a.pushSheet(sh)
	a.Sheet.Status = fmt.Sprintf("pivot %s", a.Sheet.Name)
}

func (a *App) openDedupeSheet() {
	sh, err := a.Sheet.DedupeSheet(a.Sheet.CursorCol)
	if err != nil {
		a.Sheet.Status = fmt.Sprintf("dedupe failed: %v", err)
		return
	}
	a.pushSheet(sh)
	a.Sheet.Status = fmt.Sprintf("dedupe %s", a.Sheet.Name)
}

func (a *App) openMeltSheet() {
	sh, err := a.Sheet.MeltSheet(a.Sheet.CursorCol)
	if err != nil {
		a.Sheet.Status = fmt.Sprintf("melt failed: %v", err)
		return
	}
	a.pushSheet(sh)
	a.Sheet.Status = fmt.Sprintf("melt %s", a.Sheet.Name)
}

func (a *App) columnsMetaTarget() (*sheet.Sheet, int, bool) {
	if a.Sheet.MetaKind != "columns" || len(a.Sheet.MetaTargets) == 0 {
		return nil, 0, false
	}
	if a.Sheet.CursorRow < 0 || a.Sheet.CursorRow >= len(a.Sheet.MetaRows) {
		return nil, 0, false
	}
	return a.Sheet.MetaTargets[0], a.Sheet.MetaRows[a.Sheet.CursorRow], true
}

func (a *App) refreshColumnsMeta(status string) {
	source, _, ok := a.columnsMetaTarget()
	if !ok {
		return
	}
	oldRow := a.Sheet.CursorRow
	oldCol := a.Sheet.CursorCol
	refreshed := source.ColumnsSheet()
	if oldRow >= len(refreshed.Rows) {
		oldRow = len(refreshed.Rows) - 1
	}
	if oldRow < 0 {
		oldRow = 0
	}
	if oldCol >= len(refreshed.Columns) {
		oldCol = len(refreshed.Columns) - 1
	}
	if oldCol < 0 {
		oldCol = 0
	}
	refreshed.SetCursorRow(oldRow)
	refreshed.SetCursorCol(oldCol)
	refreshed.Status = status
	a.stack[len(a.stack)-1] = refreshed
	a.Sheet = refreshed
	a.colWidths = columnWidths(refreshed)
	a.ensureVisible(a.size())
}

func (a *App) toggleColumnsMetaHidden() bool {
	source, columnIndex, ok := a.columnsMetaTarget()
	if !ok {
		return false
	}
	name := source.Columns[columnIndex].Name
	if err := source.ToggleHidden(columnIndex); err != nil {
		a.Sheet.Status = fmt.Sprintf("hide failed: %v", err)
		return true
	}
	a.refreshColumnsMeta(fmt.Sprintf("toggled hidden for %s", name))
	return true
}

func (a *App) showAllColumnsMeta() bool {
	source, _, ok := a.columnsMetaTarget()
	if !ok {
		return false
	}
	count := source.ShowAllColumns()
	a.refreshColumnsMeta(fmt.Sprintf("revealed %d column(s)", count))
	return true
}

func (a *App) overrideColumnsMetaType(kind sheet.ValueKind) bool {
	source, columnIndex, ok := a.columnsMetaTarget()
	if !ok {
		return false
	}
	if err := source.OverrideColumnKind(columnIndex, kind); err != nil {
		a.Sheet.Status = fmt.Sprintf("type override failed: %v", err)
		return true
	}
	a.refreshColumnsMeta(fmt.Sprintf("column %s type set to %s", source.Columns[columnIndex].Name, kind))
	return true
}

func (a *App) beginRenameInput() {
	if _, _, ok := a.columnsMetaTarget(); ok {
		a.beginInput(inputModeRename, "")
		return
	}
	a.beginInput(inputModeRename, "")
}

func (a *App) beginResizeInput() {
	if _, _, ok := a.columnsMetaTarget(); ok {
		a.beginInput(inputModeResize, "")
		return
	}
	a.beginInput(inputModeResize, "")
}

func (a *App) beginEditCellInput() {
	if len(a.Sheet.Rows) == 0 {
		a.Sheet.Status = "no cell to edit"
		return
	}
	a.beginInput(inputModeEditCell, a.Sheet.Cell(a.Sheet.CursorRow, a.Sheet.CursorCol))
}

func (a *App) buildSheetsSheet() *sheet.Sheet {
	targets := append([]*sheet.Sheet(nil), a.stack...)
	sh := sheet.New("sheets", "", []string{"depth", "name", "rows", "columns", "current"})
	sh.MetaKind = "sheets"
	sh.MetaTargets = targets
	sh.Columns[0].Kind = sheet.KindInt
	sh.Columns[2].Kind = sheet.KindInt
	sh.Columns[3].Kind = sheet.KindInt
	sh.Columns[4].Kind = sheet.KindBool

	for i, target := range targets {
		current := target == a.Sheet
		sh.AddRawRow([]string{
			fmt.Sprintf("%d", i+1),
			target.Name,
			fmt.Sprintf("%d", len(target.Rows)),
			fmt.Sprintf("%d", len(target.Columns)),
			fmt.Sprintf("%t", current),
		})
	}

	return sh
}

func (a *App) buildCommandsSheet() *sheet.Sheet {
	sh := sheet.New("commands", "", []string{"keys", "name", "help"})
	sh.MetaKind = "commands"
	for _, command := range commandCatalog {
		sh.AddRawRow([]string{command.Keys, command.Name, command.Help})
	}
	return sh
}

func (a *App) buildCommandLogSheet() *sheet.Sheet {
	sh := sheet.New("cmdlog", "", []string{"index", "keys", "name", "sheet", "detail"})
	sh.MetaKind = "cmdlog"
	sh.Columns[0].Kind = sheet.KindInt
	for _, entry := range a.commandLog {
		sh.AddRawRow([]string{
			fmt.Sprintf("%d", entry.Index),
			entry.Keys,
			entry.Name,
			entry.Sheet,
			entry.Detail,
		})
	}
	return sh
}

func (a *App) recordCommand(keys, name, detail string) {
	entry := commandLogEntry{
		Index:  len(a.commandLog) + 1,
		Keys:   keys,
		Name:   name,
		Sheet:  a.Sheet.Name,
		Detail: detail,
	}
	a.commandLog = append(a.commandLog, entry)
}

func (a *App) undoTarget() (*sheet.Sheet, bool) {
	if a.Sheet.MetaKind == "columns" && len(a.Sheet.MetaTargets) > 0 {
		return a.Sheet.MetaTargets[0], true
	}
	return a.Sheet, false
}

func (a *App) undoCurrentSheet() {
	target, refreshColumnsMeta := a.undoTarget()
	label, ok := target.Undo()
	if !ok {
		a.Sheet.Status = "nothing to undo"
		return
	}
	if refreshColumnsMeta {
		a.refreshColumnsMeta(fmt.Sprintf("undid %s", label))
		a.recordCommand("U", "undo", fmt.Sprintf("undid %s", label))
		return
	}
	a.colWidths = columnWidths(target)
	target.Status = fmt.Sprintf("undid %s", label)
	a.recordCommand("U", "undo", target.Status)
}

func (a *App) redoCurrentSheet() {
	target, refreshColumnsMeta := a.undoTarget()
	label, ok := target.Redo()
	if !ok {
		a.Sheet.Status = "nothing to redo"
		return
	}
	if refreshColumnsMeta {
		a.refreshColumnsMeta(fmt.Sprintf("redid %s", label))
		a.recordCommand("R", "redo", fmt.Sprintf("redid %s", label))
		return
	}
	a.colWidths = columnWidths(target)
	target.Status = fmt.Sprintf("redid %s", label)
	a.recordCommand("R", "redo", target.Status)
}

func (a *App) activateCurrentRow() {
	switch a.Sheet.MetaKind {
	case "sheets":
		if a.Sheet.CursorRow < 0 || a.Sheet.CursorRow >= len(a.Sheet.MetaTargets) {
			return
		}
		target := a.Sheet.MetaTargets[a.Sheet.CursorRow]
		for i, candidate := range a.stack {
			if candidate != target {
				continue
			}
			a.stack = a.stack[:i+1]
			a.Sheet = candidate
			a.RowOffset = 0
			a.ColOffset = 0
			a.mode = inputModeNone
			a.inputValue = nil
			a.colWidths = columnWidths(a.Sheet)
			a.Sheet.Status = fmt.Sprintf("switched to %s", a.Sheet.Name)
			return
		}
	case "freq":
		a.openCurrentFrequencyValueSheet()
		return
	default:
		a.openCurrentRowSheet()
		return
	}
}

func (a *App) openCurrentRowSheet() {
	if len(a.Sheet.Rows) == 0 {
		a.Sheet.Status = "no row to open"
		return
	}
	a.pushSheet(a.Sheet.RowDetailsSheet(a.Sheet.CursorRow))
	a.Sheet.Status = fmt.Sprintf("row %s", a.Sheet.Name)
}

func (a *App) openCurrentFrequencyValueSheet() {
	if len(a.Sheet.Rows) == 0 {
		a.Sheet.Status = "no frequency row to open"
		return
	}
	if len(a.Sheet.MetaTargets) == 0 || len(a.Sheet.MetaCols) == 0 {
		a.openCurrentRowSheet()
		return
	}

	source := a.Sheet.MetaTargets[0]
	columnIndex := a.Sheet.MetaCols[0]
	value := a.Sheet.Cell(a.Sheet.CursorRow, 0)
	filtered, err := source.FilterValueSheet(columnIndex, value)
	if err != nil {
		a.Sheet.Status = fmt.Sprintf("filter failed: %v", err)
		return
	}
	a.pushSheet(filtered)
	a.Sheet.Status = fmt.Sprintf("filter %s=%s", source.Columns[columnIndex].Name, value)
}

func (a *App) handleInputKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape:
		if a.mode == inputModeSearch {
			a.Sheet.ClearSearch()
		}
		a.mode = inputModeNone
		a.inputValue = nil
	case tcell.KeyEnter:
		a.commitInput()
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(a.inputValue) > 0 {
			a.inputValue = a.inputValue[:len(a.inputValue)-1]
		}
	case tcell.KeyRune:
		a.inputValue = append(a.inputValue, ev.Rune())
	}

	a.ensureVisible(a.size())
	return false
}

func (a *App) pageSize() int {
	_, height := a.size()
	if rows := height - 4; rows > 0 {
		return rows
	}
	return 1
}

func (a *App) size() (int, int) {
	if a.Screen == nil {
		return 80, 24
	}
	return a.Screen.Size()
}

func (a *App) saveSuggested() {
	path, err := a.Sheet.SaveSuggested()
	if err != nil {
		a.Sheet.Status = fmt.Sprintf("save failed: %v", err)
		return
	}
	a.Sheet.Status = fmt.Sprintf("saved %s", path)
}

func (a *App) beginInput(mode inputMode, initial string) {
	a.mode = mode
	a.inputValue = []rune(initial)
}

func (a *App) commitInput() {
	value := string(a.inputValue)
	mode := a.mode
	a.mode = inputModeNone
	a.inputValue = nil

	switch mode {
	case inputModeSearch:
		a.Sheet.Search(value)
		a.recordCommand("/", "search", fmt.Sprintf("search %q", value))
	case inputModeEditCell:
		if err := a.Sheet.SetCell(a.Sheet.CursorRow, a.Sheet.CursorCol, value); err != nil {
			a.Sheet.Status = fmt.Sprintf("edit failed: %v", err)
			return
		}
		a.colWidths = columnWidths(a.Sheet)
		a.Sheet.Status = fmt.Sprintf("edited %s row %d", a.Sheet.Columns[a.Sheet.CursorCol].Name, a.Sheet.CursorRow+1)
		a.recordCommand("e", "edit-cell", a.Sheet.Status)
	case inputModeExpr:
		name, err := a.Sheet.AddExprColumn(value)
		if err != nil {
			a.Sheet.Status = fmt.Sprintf("expression failed: %v", err)
			return
		}
		a.colWidths = columnWidths(a.Sheet)
		a.Sheet.Status = fmt.Sprintf("added expression column %s", name)
		a.recordCommand("=", "expr-col", a.Sheet.Status)
	case inputModeRename:
		if source, columnIndex, ok := a.columnsMetaTarget(); ok {
			if err := source.RenameColumn(columnIndex, value); err != nil {
				a.Sheet.Status = fmt.Sprintf("rename failed: %v", err)
				return
			}
			a.refreshColumnsMeta(fmt.Sprintf("renamed column to %s", source.Columns[columnIndex].Name))
			a.recordCommand("^", "rename-col", a.Sheet.Status)
			return
		}
		if err := a.Sheet.RenameColumn(a.Sheet.CursorCol, value); err != nil {
			a.Sheet.Status = fmt.Sprintf("rename failed: %v", err)
			return
		}
		a.colWidths = columnWidths(a.Sheet)
		a.Sheet.Status = fmt.Sprintf("renamed column to %s", a.Sheet.Columns[a.Sheet.CursorCol].Name)
		a.recordCommand("^", "rename-col", a.Sheet.Status)
	case inputModeResize:
		width, err := sheet.ParseWidth(value)
		if err != nil {
			a.Sheet.Status = fmt.Sprintf("resize failed: %v", err)
			return
		}
		if source, columnIndex, ok := a.columnsMetaTarget(); ok {
			if err := source.SetColumnWidth(columnIndex, width); err != nil {
				a.Sheet.Status = fmt.Sprintf("resize failed: %v", err)
				return
			}
			a.refreshColumnsMeta(fmt.Sprintf("set width %d for %s", width, source.Columns[columnIndex].Name))
			a.recordCommand("_", "resize-col", a.Sheet.Status)
			return
		}
		if err := a.Sheet.SetColumnWidth(a.Sheet.CursorCol, width); err != nil {
			a.Sheet.Status = fmt.Sprintf("resize failed: %v", err)
			return
		}
		a.colWidths = columnWidths(a.Sheet)
		a.Sheet.Status = fmt.Sprintf("set width %d for %s", width, a.Sheet.Columns[a.Sheet.CursorCol].Name)
		a.recordCommand("_", "resize-col", a.Sheet.Status)
	case inputModeRegexSelect:
		count, err := a.Sheet.SelectByRegex(a.Sheet.CursorCol, value, false)
		if err != nil {
			a.Sheet.Status = fmt.Sprintf("select failed: %v", err)
			return
		}
		a.Sheet.Status = fmt.Sprintf("selected %d row(s)", count)
		a.recordCommand("|", "regex-select", a.Sheet.Status)
	case inputModeRegexUnselect:
		count, err := a.Sheet.UnselectByRegex(a.Sheet.CursorCol, value, false)
		if err != nil {
			a.Sheet.Status = fmt.Sprintf("unselect failed: %v", err)
			return
		}
		a.Sheet.Status = fmt.Sprintf("unselected %d row(s)", count)
		a.recordCommand("\\", "regex-unselect", a.Sheet.Status)
	}
}

func (a *App) overrideColumnType(kind sheet.ValueKind) {
	if err := a.Sheet.OverrideColumnKind(a.Sheet.CursorCol, kind); err != nil {
		a.Sheet.Status = fmt.Sprintf("type override failed: %v", err)
		return
	}
	a.colWidths = columnWidths(a.Sheet)
	a.Sheet.Status = fmt.Sprintf("column %s type set to %s", a.Sheet.Columns[a.Sheet.CursorCol].Name, kind)
	a.recordCommand(string(kindKey(kind)), "type-override", a.Sheet.Status)
}

func kindKey(kind sheet.ValueKind) rune {
	switch kind {
	case sheet.KindString:
		return '~'
	case sheet.KindInt:
		return '#'
	case sheet.KindFloat:
		return '%'
	case sheet.KindCurrency:
		return '$'
	case sheet.KindDate:
		return '@'
	default:
		return '?'
	}
}

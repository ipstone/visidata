package tui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/ipstone/visidata/go-visidata/pkg/sheet"
)

type App struct {
	Screen      tcell.Screen
	Sheet       *sheet.Sheet
	RowOffset   int
	ColOffset   int
	searchMode  bool
	searchQuery []rune
	colWidths   []int
}

func New(sh *sheet.Sheet, screen tcell.Screen) *App {
	return &App{
		Screen:    screen,
		Sheet:     sh,
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
	if a.searchMode {
		return a.handleSearchKey(ev)
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
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'q':
			return true
		case '/':
			a.searchMode = true
			a.searchQuery = []rune(a.Sheet.SearchState.Query)
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
			}
		case 'S':
			a.saveSuggested()
		}
	}

	a.ensureVisible(a.size())
	return false
}

func (a *App) handleSearchKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyEscape:
		a.searchMode = false
		a.searchQuery = nil
		a.Sheet.ClearSearch()
	case tcell.KeyEnter:
		a.searchMode = false
		a.Sheet.Search(string(a.searchQuery))
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(a.searchQuery) > 0 {
			a.searchQuery = a.searchQuery[:len(a.searchQuery)-1]
		}
	case tcell.KeyRune:
		a.searchQuery = append(a.searchQuery, ev.Rune())
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

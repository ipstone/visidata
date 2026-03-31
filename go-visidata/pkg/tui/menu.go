package tui

import "github.com/gdamore/tcell/v2"

type menuGroup struct {
	Label string
	Items []menuItem
}

type menuItem struct {
	Label   string
	Command string
}

var menuCatalog = []menuGroup{
	{
		Label: "File",
		Items: []menuItem{
			{Label: "Save", Command: "save"},
			{Label: "Quit", Command: "quit"},
		},
	},
	{
		Label: "Edit",
		Items: []menuItem{
			{Label: "Undo", Command: "undo"},
			{Label: "Redo", Command: "redo"},
			{Label: "Start/Stop Macro", Command: "macro-record"},
			{Label: "Replay Macro", Command: "macro-play"},
			{Label: "Edit Cell", Command: "edit-cell"},
			{Label: "Rename Column", Command: "rename-col"},
			{Label: "Resize Column", Command: "resize-col"},
		},
	},
	{
		Label: "Data",
		Items: []menuItem{
			{Label: "Search", Command: "search"},
			{Label: "Expression Column", Command: "expr-col"},
			{Label: "Sort Asc", Command: "sort-asc"},
			{Label: "Sort Desc", Command: "sort-desc"},
			{Label: "Frequency", Command: "frequency"},
			{Label: "Graph", Command: "graph"},
			{Label: "Sparkline Column", Command: "sparkline-col"},
			{Label: "Describe", Command: "describe"},
			{Label: "Pivot", Command: "pivot"},
			{Label: "Transpose", Command: "transpose"},
			{Label: "Melt", Command: "melt"},
			{Label: "Join", Command: "join"},
			{Label: "Dedupe", Command: "dedupe"},
		},
	},
	{
		Label: "View",
		Items: []menuItem{
			{Label: "Sheets", Command: "sheets"},
			{Label: "Sidebar", Command: "sidebar"},
			{Label: "Columns", Command: "columns"},
			{Label: "Options", Command: "options"},
			{Label: "Command Log", Command: "cmdlog"},
			{Label: "Show Commands", Command: "commands"},
			{Label: "Hide Column", Command: "hide-column"},
			{Label: "Show All Columns", Command: "show-all-columns"},
			{Label: "Regex Capture Columns", Command: "regex-capture"},
			{Label: "Regex Split Columns", Command: "regex-split"},
		},
	},
	{
		Label: "Help",
		Items: []menuItem{
			{Label: "Command Palette", Command: "command-palette"},
			{Label: "Command Reference", Command: "commands"},
		},
	},
}

func (a *App) beginMenu() {
	a.mode = inputModeMenu
	a.inputValue = nil
	a.paletteIndex = 0
	if len(menuCatalog) == 0 {
		a.menuIndex = 0
		a.menuItemIndex = 0
		return
	}
	if a.menuIndex >= len(menuCatalog) {
		a.menuIndex = 0
	}
	if a.menuIndex < 0 {
		a.menuIndex = 0
	}
	a.clampMenuItemIndex()
}

func (a *App) handleMenuKey(ev *tcell.EventKey) bool {
	if len(menuCatalog) == 0 {
		a.mode = inputModeNone
		return false
	}

	switch ev.Key() {
	case tcell.KeyEscape:
		a.mode = inputModeNone
	case tcell.KeyLeft:
		a.menuIndex = (a.menuIndex - 1 + len(menuCatalog)) % len(menuCatalog)
		a.clampMenuItemIndex()
	case tcell.KeyRight:
		a.menuIndex = (a.menuIndex + 1) % len(menuCatalog)
		a.clampMenuItemIndex()
	case tcell.KeyUp:
		items := menuCatalog[a.menuIndex].Items
		if len(items) > 0 {
			a.menuItemIndex = (a.menuItemIndex - 1 + len(items)) % len(items)
		}
	case tcell.KeyDown:
		items := menuCatalog[a.menuIndex].Items
		if len(items) > 0 {
			a.menuItemIndex = (a.menuItemIndex + 1) % len(items)
		}
	case tcell.KeyEnter:
		items := menuCatalog[a.menuIndex].Items
		if len(items) == 0 {
			a.mode = inputModeNone
			return false
		}
		command := items[a.menuItemIndex].Command
		a.mode = inputModeNone
		return a.executeCommand(command)
	case tcell.KeyRune:
		r := ev.Rune()
		for index, group := range menuCatalog {
			if len(group.Label) > 0 && rune(group.Label[0]) == r {
				a.menuIndex = index
				a.clampMenuItemIndex()
				return false
			}
		}
	}

	return false
}

func (a *App) clampMenuItemIndex() {
	if len(menuCatalog) == 0 {
		a.menuIndex = 0
		a.menuItemIndex = 0
		return
	}
	if a.menuIndex < 0 {
		a.menuIndex = 0
	}
	if a.menuIndex >= len(menuCatalog) {
		a.menuIndex = len(menuCatalog) - 1
	}
	items := menuCatalog[a.menuIndex].Items
	if len(items) == 0 {
		a.menuItemIndex = 0
		return
	}
	if a.menuItemIndex < 0 {
		a.menuItemIndex = 0
	}
	if a.menuItemIndex >= len(items) {
		a.menuItemIndex = len(items) - 1
	}
}

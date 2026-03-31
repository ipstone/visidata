package tui

import "github.com/gdamore/tcell/v2"

const minSidebarWidth = 20

func (a *App) toggleSidebar() {
	a.sidebarVisible = !a.sidebarVisible
	if !a.sidebarVisible {
		return
	}
	if len(a.stack) == 0 {
		a.sidebarIndex = 0
		return
	}
	a.sidebarIndex = len(a.stack) - 1
}

func (a *App) closeSidebar() {
	a.sidebarVisible = false
}

func (a *App) handleSidebarKey(ev *tcell.EventKey) bool {
	switch ev.Key() {
	case tcell.KeyCtrlC:
		return true
	case tcell.KeyEscape, tcell.KeyTab, tcell.KeyRight:
		a.closeSidebar()
	case tcell.KeyUp:
		if a.sidebarIndex > 0 {
			a.sidebarIndex--
		}
	case tcell.KeyDown:
		if a.sidebarIndex < len(a.stack)-1 {
			a.sidebarIndex++
		}
	case tcell.KeyEnter:
		a.activateSidebarSelection()
		a.closeSidebar()
	case tcell.KeyRune:
		switch ev.Rune() {
		case 'j':
			if a.sidebarIndex < len(a.stack)-1 {
				a.sidebarIndex++
			}
		case 'k':
			if a.sidebarIndex > 0 {
				a.sidebarIndex--
			}
		}
	}
	return false
}

func (a *App) activateSidebarSelection() {
	if a.sidebarIndex < 0 || a.sidebarIndex >= len(a.stack) {
		return
	}
	target := a.stack[a.sidebarIndex]
	a.stack = a.stack[:a.sidebarIndex+1]
	a.Sheet = target
	a.RowOffset = 0
	a.ColOffset = 0
	a.colWidths = columnWidths(a.Sheet)
	a.Sheet.Status = "switched via sidebar"
}

func (a *App) sidebarWidth(totalWidth int) int {
	if !a.sidebarVisible || totalWidth < minSidebarWidth*2 {
		return 0
	}
	return min(28, max(minSidebarWidth, totalWidth/4))
}

func (a *App) contentFrame(totalWidth int) (int, int) {
	sidebarWidth := a.sidebarWidth(totalWidth)
	return sidebarWidth, max(0, totalWidth-sidebarWidth)
}

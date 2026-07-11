package tui

func (a *App) toggleMacroRecording() {
	if a.macroRecording {
		a.macroRecording = false
		a.Sheet.Status = "stopped macro recording"
		a.recordCommand("zz", "macro-record", a.Sheet.Status)
		return
	}

	a.macro = nil
	a.macroRecording = true
	a.Sheet.Status = "recording macro"
	a.recordCommand("zz", "macro-record", a.Sheet.Status)
}

func (a *App) replayMacro() bool {
	if len(a.macro) == 0 {
		a.Sheet.Status = "no macro recorded"
		return false
	}

	a.macroPlaying = true
	defer func() {
		a.macroPlaying = false
	}()

	for _, command := range a.macro {
		if mode, ok := macroInputMode(command.Name); ok {
			a.beginInput(mode, command.Input)
			a.commitInput()
			continue
		}
		if shouldQuit := a.executeCommand(command.Name); shouldQuit {
			return true
		}
	}

	a.Sheet.Status = "replayed macro"
	a.recordCommand("zZ", "macro-play", a.Sheet.Status)
	return false
}

func (a *App) captureMacro(name, input string) {
	if !a.macroRecording || a.macroPlaying || shouldSkipMacro(name) {
		return
	}
	a.macro = append(a.macro, macroCommand{Name: name, Input: input})
}

func shouldSkipMacro(name string) bool {
	switch name {
	case "", "command-palette", "macro-record", "macro-play", "quit", "commands", "cmdlog":
		return true
	default:
		return false
	}
}

func isInputBackedCommand(name string) bool {
	_, ok := macroInputMode(name)
	return ok
}

func macroInputMode(name string) (inputMode, bool) {
	switch name {
	case "search":
		return inputModeSearch, true
	case "goto-col-regex":
		return inputModeGotoColRegex, true
	case "goto-row-regex":
		return inputModeGotoRowRegex, true
	case "goto-col-number":
		return inputModeGotoColNumber, true
	case "goto-row-number":
		return inputModeGotoRowNumber, true
	case "edit-cell":
		return inputModeEditCell, true
	case "expr-col":
		return inputModeExpr, true
	case "rename-col":
		return inputModeRename, true
	case "resize-col":
		return inputModeResize, true
	case "regex-capture":
		return inputModeRegexCapture, true
	case "regex-split":
		return inputModeRegexSplit, true
	case "regex-select":
		return inputModeRegexSelect, true
	case "regex-unselect":
		return inputModeRegexUnselect, true
	default:
		return inputModeNone, false
	}
}

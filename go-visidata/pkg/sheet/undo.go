package sheet

type sheetSnapshot struct {
	Columns     []Column
	Rows        []Row
	MetaKind    string
	MetaCols    []int
	MetaRows    []int
	MetaTargets []*Sheet
	CursorRow   int
	CursorCol   int
	Selected    []bool
	SortState   SortState
	SearchState SearchState
	Clipboard   Clipboard
	Status      string
	RowIDs      []int
	NextRowID   int
}

type sheetUndoEntry struct {
	label    string
	snapshot sheetSnapshot
}

func (s *Sheet) pushUndo(label string) {
	if s == nil {
		return
	}
	s.undoStack = append(s.undoStack, sheetUndoEntry{label: label, snapshot: s.snapshot()})
	s.redoStack = nil
}

func (s *Sheet) Undo() (string, bool) {
	if s == nil || len(s.undoStack) == 0 {
		return "", false
	}
	entry := s.undoStack[len(s.undoStack)-1]
	s.undoStack = s.undoStack[:len(s.undoStack)-1]
	s.redoStack = append(s.redoStack, sheetUndoEntry{label: entry.label, snapshot: s.snapshot()})
	s.restoreSnapshot(entry.snapshot)
	return entry.label, true
}

func (s *Sheet) Redo() (string, bool) {
	if s == nil || len(s.redoStack) == 0 {
		return "", false
	}
	entry := s.redoStack[len(s.redoStack)-1]
	s.redoStack = s.redoStack[:len(s.redoStack)-1]
	s.undoStack = append(s.undoStack, sheetUndoEntry{label: entry.label, snapshot: s.snapshot()})
	s.restoreSnapshot(entry.snapshot)
	return entry.label, true
}

func (s *Sheet) UndoDepth() int {
	return len(s.undoStack)
}

func (s *Sheet) RedoDepth() int {
	return len(s.redoStack)
}

func (s *Sheet) snapshot() sheetSnapshot {
	columns := append([]Column(nil), s.Columns...)
	rows := make([]Row, len(s.Rows))
	for i, row := range s.Rows {
		rows[i] = append(Row(nil), row...)
	}
	selected := append([]bool(nil), s.Selected...)
	metaCols := append([]int(nil), s.MetaCols...)
	metaRows := append([]int(nil), s.MetaRows...)
	metaTargets := append([]*Sheet(nil), s.MetaTargets...)
	rowIDs := append([]int(nil), s.rowIDs...)
	matches := append([]Position(nil), s.SearchState.Matches...)

	return sheetSnapshot{
		Columns:     columns,
		Rows:        rows,
		MetaKind:    s.MetaKind,
		MetaCols:    metaCols,
		MetaRows:    metaRows,
		MetaTargets: metaTargets,
		CursorRow:   s.CursorRow,
		CursorCol:   s.CursorCol,
		Selected:    selected,
		SortState:   s.SortState,
		SearchState: SearchState{Query: s.SearchState.Query, Matches: matches, CurrentMatch: s.SearchState.CurrentMatch},
		Clipboard:   s.Clipboard,
		Status:      s.Status,
		RowIDs:      rowIDs,
		NextRowID:   s.nextRowID,
	}
}

func (s *Sheet) restoreSnapshot(snapshot sheetSnapshot) {
	s.Columns = append([]Column(nil), snapshot.Columns...)
	s.Rows = make([]Row, len(snapshot.Rows))
	for i, row := range snapshot.Rows {
		s.Rows[i] = append(Row(nil), row...)
	}
	s.MetaKind = snapshot.MetaKind
	s.MetaCols = append([]int(nil), snapshot.MetaCols...)
	s.MetaRows = append([]int(nil), snapshot.MetaRows...)
	s.MetaTargets = append([]*Sheet(nil), snapshot.MetaTargets...)
	s.CursorRow = snapshot.CursorRow
	s.CursorCol = snapshot.CursorCol
	s.Selected = append([]bool(nil), snapshot.Selected...)
	s.SortState = snapshot.SortState
	s.SearchState = SearchState{
		Query:        snapshot.SearchState.Query,
		Matches:      append([]Position(nil), snapshot.SearchState.Matches...),
		CurrentMatch: snapshot.SearchState.CurrentMatch,
	}
	s.Clipboard = snapshot.Clipboard
	s.Status = snapshot.Status
	s.rowIDs = append([]int(nil), snapshot.RowIDs...)
	s.nextRowID = snapshot.NextRowID
	s.clampCursor()
}

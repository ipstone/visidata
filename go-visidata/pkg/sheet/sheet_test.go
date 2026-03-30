package sheet

import "testing"

func TestInferColumnKinds(t *testing.T) {
	sh := New("test.csv", "/tmp/test.csv", []string{"id", "price", "created_at", "label"})
	sh.AddRow([]string{"1", "12.50", "2026-03-30", "alpha"})
	sh.AddRow([]string{"2", "99.00", "2026-03-31", "beta"})
	sh.InferColumnKinds()

	want := []ValueKind{KindInt, KindFloat, KindDate, KindString}
	for i, col := range sh.Columns {
		if col.Kind != want[i] {
			t.Fatalf("column %d kind = %s, want %s", i, col.Kind, want[i])
		}
	}
}

func TestSummary(t *testing.T) {
	sh := New("data.csv", "/tmp/data.csv", []string{"a", "b"})
	sh.AddRow([]string{"1", "2"})
	if got, want := sh.Summary(), "data.csv: 1 row(s) x 2 column(s)"; got != want {
		t.Fatalf("summary = %q, want %q", got, want)
	}
}

func TestLooksLikeDate(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{value: "2026-03-30", want: true},
		{value: "2026-03-30 17:59:44", want: true},
		{value: "7/3/2018 1:47p", want: true},
		{value: "7/3/2018 1:47pm", want: true},
		{value: "7/3/2018 1:47PM", want: true},
		{value: "not-a-date", want: false},
	}

	for _, tc := range tests {
		if got := looksLikeDate(tc.value); got != tc.want {
			t.Fatalf("looksLikeDate(%q) = %v, want %v", tc.value, got, tc.want)
		}
	}
}

func TestCursorMovementClampsToSheetBounds(t *testing.T) {
	sh := New("data.csv", "/tmp/data.csv", []string{"a", "b"})
	sh.AddRow([]string{"1", "2"})
	sh.AddRow([]string{"3", "4"})

	sh.MoveCursorRow(10)
	sh.MoveCursorCol(10)
	if sh.CursorRow != 1 || sh.CursorCol != 1 {
		t.Fatalf("cursor = (%d,%d), want (1,1)", sh.CursorRow, sh.CursorCol)
	}

	sh.MoveCursorRow(-10)
	sh.MoveCursorCol(-10)
	if sh.CursorRow != 0 || sh.CursorCol != 0 {
		t.Fatalf("cursor = (%d,%d), want (0,0)", sh.CursorRow, sh.CursorCol)
	}
}

func TestCellReturnsEmptyOutsideBounds(t *testing.T) {
	sh := New("data.csv", "/tmp/data.csv", []string{"a"})
	sh.AddRow([]string{"1"})

	for _, tc := range []struct {
		row, col int
		want     string
	}{
		{row: 0, col: 0, want: "1"},
		{row: -1, col: 0, want: ""},
		{row: 0, col: 1, want: ""},
		{row: 1, col: 0, want: ""},
	} {
		if got := sh.Cell(tc.row, tc.col); got != tc.want {
			t.Fatalf("Cell(%d,%d) = %q, want %q", tc.row, tc.col, got, tc.want)
		}
	}
}

func TestToggleSortAndClearSort(t *testing.T) {
	sh := New("people.csv", "/tmp/people.csv", []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.AddRow([]string{"Bob", "20"})
	sh.AddRow([]string{"Carol", "10"})
	sh.InferColumnKinds()

	sh.ToggleSelected(1)
	sh.ToggleSort(1, SortAsc)
	if got := sh.Cell(0, 0); got != "Carol" {
		t.Fatalf("first row after asc sort = %q, want Carol", got)
	}
	if !sh.IsSelected(1) || sh.Cell(1, 0) != "Bob" {
		t.Fatalf("expected selected row to move with sorted data, got row %q selected=%v", sh.Cell(1, 0), sh.IsSelected(1))
	}

	sh.ToggleSort(1, SortDesc)
	if got := sh.Cell(0, 0); got != "Alice" {
		t.Fatalf("first row after desc sort = %q, want Alice", got)
	}

	sh.ClearSort()
	if got := sh.Cell(0, 0); got != "Alice" {
		t.Fatalf("first row after clear sort = %q, want Alice", got)
	}
	if sh.SortState.Direction != SortNone {
		t.Fatalf("sort state = %+v, want cleared", sh.SortState)
	}
}

func TestSearchAndNextMatch(t *testing.T) {
	sh := New("people.csv", "/tmp/people.csv", []string{"name", "city"})
	sh.AddRow([]string{"Alice", "Tokyo"})
	sh.AddRow([]string{"Bob", "Osaka"})
	sh.AddRow([]string{"Carol", "Kyoto"})
	sh.InferColumnKinds()

	if got := sh.Search("o"); got != 5 {
		t.Fatalf("Search returned %d matches, want 5", got)
	}
	if sh.CursorRow != 0 || sh.CursorCol != 1 {
		t.Fatalf("cursor after search = (%d,%d), want (0,1)", sh.CursorRow, sh.CursorCol)
	}
	if !sh.NextMatch(1) || sh.CursorRow != 1 {
		t.Fatalf("next match did not advance to row 1, got (%d,%d)", sh.CursorRow, sh.CursorCol)
	}
	if !sh.IsSearchMatch(2, 1) {
		t.Fatal("expected Kyoto cell to be marked as a search match")
	}

	sh.ClearSearch()
	if sh.SearchState.Query != "" || len(sh.SearchState.Matches) != 0 {
		t.Fatalf("search state = %+v, want cleared", sh.SearchState)
	}
}

func TestSelectionHelpers(t *testing.T) {
	sh := New("people.csv", "/tmp/people.csv", []string{"name"})
	sh.AddRow([]string{"Alice"})
	sh.AddRow([]string{"Bob"})

	sh.ToggleSelected(0)
	if got := sh.SelectedCount(); got != 1 {
		t.Fatalf("SelectedCount = %d, want 1", got)
	}

	sh.SelectAll()
	if got := sh.SelectedCount(); got != 2 {
		t.Fatalf("SelectedCount after SelectAll = %d, want 2", got)
	}

	sh.ClearSelection()
	if got := sh.SelectedCount(); got != 0 {
		t.Fatalf("SelectedCount after ClearSelection = %d, want 0", got)
	}
}

func TestDeleteSelectedRowsOrCurrentRemovesRowsAndRefreshesSearch(t *testing.T) {
	sh := New("people.csv", "/tmp/people.csv", []string{"name", "city"})
	sh.AddRow([]string{"Alice", "Tokyo"})
	sh.AddRow([]string{"Bob", "Osaka"})
	sh.AddRow([]string{"Carol", "Kyoto"})
	sh.Search("o")
	sh.ToggleSelected(0)
	sh.ToggleSelected(2)

	if got := sh.DeleteSelectedRowsOrCurrent(); got != 2 {
		t.Fatalf("DeleteSelectedRowsOrCurrent = %d, want 2", got)
	}
	if len(sh.Rows) != 1 || sh.Cell(0, 0) != "Bob" {
		t.Fatalf("rows after delete = %#v, want only Bob", sh.Rows)
	}
	if got := sh.Clipboard.Content; got != "Alice\tTokyo\nCarol\tKyoto" {
		t.Fatalf("clipboard after delete = %q, want deleted rows", got)
	}
	if sh.SearchState.Query != "o" || len(sh.SearchState.Matches) != 2 {
		t.Fatalf("search state after delete = %+v, want refreshed matches for Bob only", sh.SearchState)
	}
}

func TestDeleteSelectedRowsOrCurrentFallsBackToCursorRow(t *testing.T) {
	sh := New("people.csv", "/tmp/people.csv", []string{"name"})
	sh.AddRow([]string{"Alice"})
	sh.AddRow([]string{"Bob"})
	sh.SetCursorRow(1)

	if got := sh.DeleteSelectedRowsOrCurrent(); got != 1 {
		t.Fatalf("DeleteSelectedRowsOrCurrent = %d, want 1", got)
	}
	if len(sh.Rows) != 1 || sh.Cell(0, 0) != "Alice" {
		t.Fatalf("rows after delete = %#v, want only Alice", sh.Rows)
	}
	if sh.CursorRow != 0 {
		t.Fatalf("CursorRow after delete = %d, want 0", sh.CursorRow)
	}
}

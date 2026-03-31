package sheet

import "testing"

func TestUndoRestoresDeletedRows(t *testing.T) {
	sh := New("people.csv", "/tmp/people.csv", []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.AddRow([]string{"Bob", "20"})
	sh.AddRow([]string{"Carol", "10"})
	sh.ToggleSelected(1)
	sh.Search("o")

	if got := sh.DeleteSelectedRowsOrCurrent(); got != 1 {
		t.Fatalf("DeleteSelectedRowsOrCurrent = %d, want 1", got)
	}
	if len(sh.Rows) != 2 {
		t.Fatalf("len(rows) after delete = %d, want 2", len(sh.Rows))
	}
	label, ok := sh.Undo()
	if !ok || label != "delete rows" {
		t.Fatalf("Undo() = (%q, %v), want (delete rows, true)", label, ok)
	}
	if len(sh.Rows) != 3 || sh.Cell(1, 0) != "Bob" {
		t.Fatalf("rows after undo = %#v, want Bob restored", sh.Rows)
	}
	if sh.SearchState.Query != "o" || len(sh.SearchState.Matches) == 0 {
		t.Fatalf("search state after undo = %+v, want restored", sh.SearchState)
	}
}

func TestUndoRestoresColumnChangesAndExprColumns(t *testing.T) {
	sh := New("orders.csv", "/tmp/orders.csv", []string{"qty", "price"})
	sh.AddRow([]string{"2", "1.5"})
	sh.AddRow([]string{"4", "2.0"})
	sh.InferColumnKinds()

	if _, err := sh.AddExprColumn("total := qty * price"); err != nil {
		t.Fatalf("AddExprColumn returned error: %v", err)
	}
	if got := len(sh.Columns); got != 3 {
		t.Fatalf("len(columns) = %d, want 3", got)
	}
	label, ok := sh.Undo()
	if !ok || label != "add expression column" {
		t.Fatalf("Undo() = (%q, %v), want (add expression column, true)", label, ok)
	}
	if got := len(sh.Columns); got != 2 {
		t.Fatalf("len(columns) after undo = %d, want 2", got)
	}

	if err := sh.RenameColumn(0, "count"); err != nil {
		t.Fatalf("RenameColumn returned error: %v", err)
	}
	if err := sh.ToggleHidden(1); err != nil {
		t.Fatalf("ToggleHidden returned error: %v", err)
	}
	if _, ok := sh.Undo(); !ok {
		t.Fatal("expected undo for hidden column toggle")
	}
	if sh.Columns[1].Hidden {
		t.Fatal("expected hidden state to be restored")
	}
	if _, ok := sh.Undo(); !ok {
		t.Fatal("expected undo for rename")
	}
	if got := sh.Columns[0].Name; got != "qty" {
		t.Fatalf("column name after undo = %q, want qty", got)
	}
}

func TestRedoReappliesDeletedRowsAndSort(t *testing.T) {
	sh := New("people.csv", "/tmp/people.csv", []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.AddRow([]string{"Bob", "20"})
	sh.AddRow([]string{"Carol", "10"})
	sh.InferColumnKinds()
	sh.SetCursorRow(1)

	if got := sh.DeleteSelectedRowsOrCurrent(); got != 1 {
		t.Fatalf("DeleteSelectedRowsOrCurrent = %d, want 1", got)
	}
	if _, ok := sh.Undo(); !ok {
		t.Fatal("expected undo to succeed")
	}
	label, ok := sh.Redo()
	if !ok || label != "delete rows" {
		t.Fatalf("Redo() = (%q, %v), want (delete rows, true)", label, ok)
	}
	if len(sh.Rows) != 2 || sh.Cell(1, 0) != "Carol" {
		t.Fatalf("rows after redo = %#v, want Bob removed again", sh.Rows)
	}

	sh = New("people.csv", "/tmp/people.csv", []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.AddRow([]string{"Bob", "20"})
	sh.AddRow([]string{"Carol", "10"})
	sh.InferColumnKinds()
	sh.ToggleSort(1, SortAsc)
	if _, ok := sh.Undo(); !ok {
		t.Fatal("expected undo for sort to succeed")
	}
	if got := sh.Cell(0, 0); got != "Alice" {
		t.Fatalf("first row after undoing sort = %q, want Alice", got)
	}
	label, ok = sh.Redo()
	if !ok || label != "sort rows" {
		t.Fatalf("Redo() = (%q, %v), want (sort rows, true)", label, ok)
	}
	if got := sh.Cell(0, 0); got != "Carol" {
		t.Fatalf("first row after redoing sort = %q, want Carol", got)
	}
}

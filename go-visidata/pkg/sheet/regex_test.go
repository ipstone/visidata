package sheet

import "testing"

func TestAddRegexCaptureColumnsUsesNamedGroups(t *testing.T) {
	sh := New("people.csv", "/tmp/people.csv", []string{"person"})
	sh.AddRow([]string{"Alice 30"})
	sh.AddRow([]string{"Bob 25"})
	sh.AddRow([]string{"invalid"})

	names, err := sh.AddRegexCaptureColumns(0, `^(?P<name>\w+)\s+(?P<age>\d+)$`)
	if err != nil {
		t.Fatalf("AddRegexCaptureColumns returned error: %v", err)
	}
	if got := names[0]; got != "name" {
		t.Fatalf("first capture name = %q, want name", got)
	}
	if got := names[1]; got != "age" {
		t.Fatalf("second capture name = %q, want age", got)
	}
	if got := len(sh.Columns); got != 3 {
		t.Fatalf("len(columns) = %d, want 3", got)
	}
	if got := sh.Cell(0, 1); got != "Alice" {
		t.Fatalf("first capture value = %q, want Alice", got)
	}
	if got := sh.Cell(1, 2); got != "25" {
		t.Fatalf("second capture age = %q, want 25", got)
	}
	if got := sh.Cell(2, 1); got != "" {
		t.Fatalf("unmatched capture value = %q, want empty", got)
	}
	if got := sh.CursorCol; got != 1 {
		t.Fatalf("cursor col = %d, want 1", got)
	}
}

func TestAddRegexSplitColumnsAppendsSplitParts(t *testing.T) {
	sh := New("coords.csv", "/tmp/coords.csv", []string{"coord"})
	sh.AddRow([]string{"10,20,30"})
	sh.AddRow([]string{"5,8"})
	sh.AddRow([]string{"1,2,3,4"})

	names, err := sh.AddRegexSplitColumns(0, `,`)
	if err != nil {
		t.Fatalf("AddRegexSplitColumns returned error: %v", err)
	}
	if got := len(names); got != 4 {
		t.Fatalf("len(names) = %d, want 4", got)
	}
	if got := names[0]; got != "coord_split1" {
		t.Fatalf("first split name = %q, want coord_split1", got)
	}
	if got := sh.Cell(0, 3); got != "30" {
		t.Fatalf("third split value = %q, want 30", got)
	}
	if got := sh.Cell(1, 4); got != "" {
		t.Fatalf("missing split value = %q, want empty", got)
	}
	if got := sh.Cell(2, 4); got != "4" {
		t.Fatalf("fourth split value = %q, want 4", got)
	}
}

func TestRegexDerivedColumnsUndoRestoresOriginalShape(t *testing.T) {
	sh := New("people.csv", "/tmp/people.csv", []string{"person"})
	sh.AddRow([]string{"Alice 30"})
	sh.AddRow([]string{"Bob 25"})

	if _, err := sh.AddRegexCaptureColumns(0, `^(\w+)\s+(\d+)$`); err != nil {
		t.Fatalf("AddRegexCaptureColumns returned error: %v", err)
	}
	if got := len(sh.Columns); got != 3 {
		t.Fatalf("len(columns) after capture = %d, want 3", got)
	}

	label, ok := sh.Undo()
	if !ok {
		t.Fatal("expected undo entry")
	}
	if label != "regex capture columns" {
		t.Fatalf("undo label = %q, want regex capture columns", label)
	}
	if got := len(sh.Columns); got != 1 {
		t.Fatalf("len(columns) after undo = %d, want 1", got)
	}
	if got := len(sh.Rows[0]); got != 1 {
		t.Fatalf("row width after undo = %d, want 1", got)
	}
}

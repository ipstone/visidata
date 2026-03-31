package sheet

import "testing"

func TestRowDetailsSheet(t *testing.T) {
	sh := New("people.csv", "/tmp/people.csv", []string{"name", "age", "city"})
	sh.AddRow([]string{"Alice", "30", "Tokyo"})
	sh.AddRow([]string{"Bob", "20", "Osaka"})
	sh.InferColumnKinds()

	row := sh.RowDetailsSheet(1)
	if got := row.Name; got != "row:people.csv:2" {
		t.Fatalf("row details name = %q, want %q", got, "row:people.csv:2")
	}
	if got := row.MetaKind; got != "row" {
		t.Fatalf("MetaKind = %q, want row", got)
	}
	if len(row.Rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3", len(row.Rows))
	}
	if got := row.Cell(0, 1); got != "name" {
		t.Fatalf("first column label = %q, want name", got)
	}
	if got := row.Cell(1, 2); got != "int" {
		t.Fatalf("age type = %q, want int", got)
	}
	if got := row.Cell(2, 3); got != "Osaka" {
		t.Fatalf("city value = %q, want Osaka", got)
	}
}

func TestColumnsSheet(t *testing.T) {
	sh := New("people.csv", "/tmp/people.csv", []string{"name", "age", "city"})
	sh.AddRow([]string{"Alice", "30", "Tokyo"})
	sh.InferColumnKinds()
	if err := sh.ToggleHidden(2); err != nil {
		t.Fatalf("ToggleHidden returned error: %v", err)
	}
	if err := sh.SetColumnWidth(1, 12); err != nil {
		t.Fatalf("SetColumnWidth returned error: %v", err)
	}

	cols := sh.ColumnsSheet()
	if got := cols.Name; got != "columns:people.csv" {
		t.Fatalf("columns sheet name = %q, want %q", got, "columns:people.csv")
	}
	if got := cols.MetaKind; got != "columns" {
		t.Fatalf("MetaKind = %q, want columns", got)
	}
	if len(cols.MetaTargets) != 1 || cols.MetaTargets[0] != sh {
		t.Fatalf("MetaTargets = %#v, want source sheet pointer", cols.MetaTargets)
	}
	if len(cols.MetaRows) != 3 || cols.MetaRows[2] != 2 {
		t.Fatalf("MetaRows = %#v, want [0 1 2]", cols.MetaRows)
	}
	if len(cols.Rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3", len(cols.Rows))
	}
	if got := cols.Cell(1, 2); got != "int" {
		t.Fatalf("age type = %q, want int", got)
	}
	if got := cols.Cell(1, 3); got != "12" {
		t.Fatalf("age width = %q, want 12", got)
	}
	if got := cols.Cell(2, 4); got != "true" {
		t.Fatalf("city hidden = %q, want true", got)
	}
	if got := cols.Cell(2, 5); got != "false" {
		t.Fatalf("city visible = %q, want false", got)
	}
}

func TestOptionsSheet(t *testing.T) {
	sh := New("people.csv", "/tmp/people.csv", []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.AddRow([]string{"Bob", "20"})
	sh.InferColumnKinds()
	sh.SetCursorRow(1)
	sh.SetCursorCol(1)
	sh.ToggleSelected(0)
	sh.ToggleSort(1, SortDesc)
	sh.Search("o")

	options := sh.OptionsSheet()
	if got := options.Name; got != "options:people.csv" {
		t.Fatalf("options sheet name = %q, want %q", got, "options:people.csv")
	}
	values := map[string]string{}
	for rowIndex := range options.Rows {
		values[options.Cell(rowIndex, 0)] = options.Cell(rowIndex, 1)
	}
	for key, want := range map[string]string{
		"rows":            "2",
		"selected_rows":   "1",
		"sort_column":     "age",
		"sort_direction":  "desc",
		"search_query":    "o",
		"search_matches":  "1",
		"cursor_row":      "2",
		"cursor_col":      "1",
		"save_path":       "/tmp/people.vdgo.csv",
		"visible_columns": "2",
	} {
		if got := values[key]; got != want {
			t.Fatalf("option %s = %q, want %q", key, got, want)
		}
	}
}

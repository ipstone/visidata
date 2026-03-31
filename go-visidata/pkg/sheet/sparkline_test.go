package sheet

import "testing"

func TestAddSparklineColumnUsesVisibleNumericColumns(t *testing.T) {
	sh := New("metrics.csv", "/tmp/metrics.csv", []string{"city", "a", "b", "c"})
	sh.AddRow([]string{"Tokyo", "1", "5", "9"})
	sh.AddRow([]string{"Osaka", "3", "3", "3"})
	sh.AddRow([]string{"Kyoto", "2", "", "8"})
	sh.InferColumnKinds()

	name, err := sh.AddSparklineColumn()
	if err != nil {
		t.Fatalf("AddSparklineColumn returned error: %v", err)
	}
	if name != "sparkline" {
		t.Fatalf("column name = %q, want sparkline", name)
	}
	if got := len(sh.Columns); got != 5 {
		t.Fatalf("len(columns) = %d, want 5", got)
	}
	if got := sh.Columns[4].Kind; got != KindString {
		t.Fatalf("sparkline kind = %s, want string", got)
	}
	if got := sh.Cell(0, 4); got != "▁▅█" {
		t.Fatalf("first sparkline = %q, want ▁▅█", got)
	}
	if got := sh.Cell(1, 4); got != "▅▅▅" {
		t.Fatalf("flat sparkline = %q, want ▅▅▅", got)
	}
	if got := sh.Cell(2, 4); got != "▁█" {
		t.Fatalf("sparse sparkline = %q, want ▁█", got)
	}
}

func TestAddSparklineColumnSkipsHiddenNumericColumns(t *testing.T) {
	sh := New("metrics.csv", "/tmp/metrics.csv", []string{"a", "b", "c"})
	sh.AddRow([]string{"1", "2", "9"})
	sh.InferColumnKinds()
	if err := sh.ToggleHidden(1); err != nil {
		t.Fatalf("ToggleHidden returned error: %v", err)
	}

	if _, err := sh.AddSparklineColumn(); err != nil {
		t.Fatalf("AddSparklineColumn returned error: %v", err)
	}
	if got := sh.Cell(0, 3); got != "▁█" {
		t.Fatalf("sparkline = %q, want ▁█", got)
	}
}

func TestAddSparklineColumnUndoRestoresOriginalShape(t *testing.T) {
	sh := New("metrics.csv", "/tmp/metrics.csv", []string{"a", "b"})
	sh.AddRow([]string{"1", "2"})
	sh.InferColumnKinds()

	if _, err := sh.AddSparklineColumn(); err != nil {
		t.Fatalf("AddSparklineColumn returned error: %v", err)
	}
	if got := len(sh.Columns); got != 3 {
		t.Fatalf("len(columns) after add = %d, want 3", got)
	}

	label, ok := sh.Undo()
	if !ok {
		t.Fatal("expected undo entry")
	}
	if label != "add sparkline column" {
		t.Fatalf("undo label = %q, want add sparkline column", label)
	}
	if got := len(sh.Columns); got != 2 {
		t.Fatalf("len(columns) after undo = %d, want 2", got)
	}
}

package sheet

import "testing"

func TestFrequencySheet(t *testing.T) {
	sh := New("people.csv", "/tmp/people.csv", []string{"city", "age"})
	sh.AddRow([]string{"Tokyo", "30"})
	sh.AddRow([]string{"Tokyo", "20"})
	sh.AddRow([]string{"Osaka", "10"})
	sh.AddRow([]string{"Tokyo", "40"})
	sh.InferColumnKinds()

	freq, err := sh.FrequencySheet(0)
	if err != nil {
		t.Fatalf("FrequencySheet returned error: %v", err)
	}
	if got := freq.Name; got != "freq:people.csv:city" {
		t.Fatalf("freq sheet name = %q, want %q", got, "freq:people.csv:city")
	}
	if len(freq.Rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(freq.Rows))
	}
	if got := freq.Cell(0, 0); got != "Tokyo" {
		t.Fatalf("top frequency value = %q, want Tokyo", got)
	}
	if got := freq.Cell(0, 1); got != "3" {
		t.Fatalf("top frequency count = %q, want 3", got)
	}
	if got := freq.Cell(0, 2); got != "75.00" {
		t.Fatalf("top frequency percent = %q, want 75.00", got)
	}
	if got := freq.Columns[1].Kind; got != KindInt {
		t.Fatalf("count kind = %s, want int", got)
	}
}

func TestDescribeSheet(t *testing.T) {
	sh := New("metrics.csv", "/tmp/metrics.csv", []string{"age", "score", "city"})
	sh.AddRow([]string{"10", "1.5", "Tokyo"})
	sh.AddRow([]string{"20", "2.5", "Osaka"})
	sh.AddRow([]string{"", "", "Tokyo"})
	sh.InferColumnKinds()

	describe := sh.DescribeSheet()
	if got := describe.Name; got != "describe:metrics.csv" {
		t.Fatalf("describe sheet name = %q, want %q", got, "describe:metrics.csv")
	}
	if len(describe.Rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3", len(describe.Rows))
	}
	if got := describe.Cell(0, 1); got != "int" {
		t.Fatalf("age type = %q, want int", got)
	}
	if got := describe.Cell(0, 3); got != "1" {
		t.Fatalf("age nulls = %q, want 1", got)
	}
	if got := describe.Cell(0, 7); got != "30" {
		t.Fatalf("age sum = %q, want 30", got)
	}
	if got := describe.Cell(0, 8); got != "15.00" {
		t.Fatalf("age avg = %q, want 15.00", got)
	}
	if got := describe.Cell(2, 4); got != "2" {
		t.Fatalf("city distinct = %q, want 2", got)
	}
}

func TestTransposeSheetUsesVisibleColumns(t *testing.T) {
	sh := New("people.csv", "/tmp/people.csv", []string{"name", "age", "city"})
	sh.AddRow([]string{"Alice", "30", "Tokyo"})
	sh.AddRow([]string{"Bob", "20", "Osaka"})
	if err := sh.ToggleHidden(1); err != nil {
		t.Fatalf("ToggleHidden returned error: %v", err)
	}

	transposed := sh.TransposeSheet()
	if got := transposed.Name; got != "transpose:people.csv" {
		t.Fatalf("transpose sheet name = %q, want %q", got, "transpose:people.csv")
	}
	if len(transposed.Columns) != 3 {
		t.Fatalf("len(columns) = %d, want 3", len(transposed.Columns))
	}
	if len(transposed.Rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(transposed.Rows))
	}
	if got := transposed.Cell(0, 0); got != "name" {
		t.Fatalf("first transposed row label = %q, want name", got)
	}
	if got := transposed.Cell(1, 2); got != "Osaka" {
		t.Fatalf("city row second value = %q, want Osaka", got)
	}
}

package sheet

import "testing"

func TestFreezeSheetCopiesVisibleColumnsAndKinds(t *testing.T) {
	sh := New("people.csv", "/tmp/people.csv", []string{"name", "age", "city"})
	sh.AddRow([]string{"Alice", "30", "Tokyo"})
	sh.AddRow([]string{"Bob", "20", "Osaka"})
	sh.InferColumnKinds()
	if err := sh.ToggleHidden(2); err != nil {
		t.Fatalf("ToggleHidden returned error: %v", err)
	}
	if err := sh.OverrideColumnKind(1, KindCurrency); err != nil {
		t.Fatalf("OverrideColumnKind returned error: %v", err)
	}

	frozen := sh.FreezeSheet()
	if got := frozen.Name; got != "freeze:people.csv" {
		t.Fatalf("freeze sheet name = %q, want %q", got, "freeze:people.csv")
	}
	if len(frozen.Columns) != 2 {
		t.Fatalf("len(columns) = %d, want 2", len(frozen.Columns))
	}
	if got := frozen.Columns[1].Kind; got != KindCurrency {
		t.Fatalf("age kind = %s, want currency", got)
	}
	if got := frozen.Cell(1, 1); got != "20" {
		t.Fatalf("frozen value = %q, want 20", got)
	}
}

func TestDedupeSheetPreservesFirstRowAndCountsDuplicates(t *testing.T) {
	sh := New("people.csv", "/tmp/people.csv", []string{"city", "age"})
	sh.AddRow([]string{"Tokyo", "30"})
	sh.AddRow([]string{"Tokyo", "20"})
	sh.AddRow([]string{"Osaka", "10"})
	sh.AddRow([]string{"Tokyo", "40"})
	sh.InferColumnKinds()

	dedupe, err := sh.DedupeSheet(0)
	if err != nil {
		t.Fatalf("DedupeSheet returned error: %v", err)
	}
	if got := dedupe.Name; got != "dedupe:people.csv:city" {
		t.Fatalf("dedupe sheet name = %q, want %q", got, "dedupe:people.csv:city")
	}
	if len(dedupe.Rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(dedupe.Rows))
	}
	if got := dedupe.Cell(0, 1); got != "30" {
		t.Fatalf("first dedupe row should keep first age, got %q", got)
	}
	if got := dedupe.Cell(0, 2); got != "3" {
		t.Fatalf("duplicate count = %q, want 3", got)
	}
	if got := dedupe.Columns[2].Kind; got != KindInt {
		t.Fatalf("duplicate_count kind = %s, want int", got)
	}
}

func TestPivotSheetGroupsByCurrentColumnAndSumsNumericColumns(t *testing.T) {
	sh := New("sales.csv", "/tmp/sales.csv", []string{"dept", "qty", "amount", "note"})
	sh.AddRow([]string{"A", "3", "1.50", "first"})
	sh.AddRow([]string{"B", "2", "2.25", "only"})
	sh.AddRow([]string{"A", "5", "3.75", "second"})
	sh.InferColumnKinds()

	pivot, err := sh.PivotSheet(0)
	if err != nil {
		t.Fatalf("PivotSheet returned error: %v", err)
	}
	if got := pivot.Name; got != "pivot:sales.csv:dept" {
		t.Fatalf("pivot sheet name = %q, want %q", got, "pivot:sales.csv:dept")
	}
	if len(pivot.Rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(pivot.Rows))
	}
	if got := pivot.Cell(0, 0); got != "A" {
		t.Fatalf("first group key = %q, want A", got)
	}
	if got := pivot.Cell(0, 1); got != "2" {
		t.Fatalf("group count = %q, want 2", got)
	}
	if got := pivot.Cell(0, 2); got != "8" {
		t.Fatalf("qty sum = %q, want 8", got)
	}
	if got := pivot.Cell(0, 3); got != "5.25" {
		t.Fatalf("amount sum = %q, want 5.25", got)
	}
	if got := len(pivot.Columns); got != 4 {
		t.Fatalf("len(columns) = %d, want 4", got)
	}
}

func TestJoinSheetMatchesRowsAndPrefixesDuplicateHeaders(t *testing.T) {
	left := New("left.csv", "/tmp/left.csv", []string{"id", "name", "score"})
	left.AddRow([]string{"1", "Alice", "10"})
	left.AddRow([]string{"2", "Bob", "20"})
	left.InferColumnKinds()

	right := New("right.csv", "/tmp/right.csv", []string{"id", "name", "city"})
	right.AddRow([]string{"1", "Alicia", "Tokyo"})
	right.AddRow([]string{"1", "Ally", "Kyoto"})
	right.AddRow([]string{"3", "Carol", "Osaka"})
	right.InferColumnKinds()

	joined, err := left.JoinSheet(right, 0, 0)
	if err != nil {
		t.Fatalf("JoinSheet returned error: %v", err)
	}
	if got := joined.Name; got != "join:left.csv+right.csv:id=id" {
		t.Fatalf("join sheet name = %q, want %q", got, "join:left.csv+right.csv:id=id")
	}
	if len(joined.Rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3", len(joined.Rows))
	}
	if got := joined.Columns[0].Name; got != "left.id" {
		t.Fatalf("first header = %q, want left.id", got)
	}
	if got := joined.Columns[3].Name; got != "right.id" {
		t.Fatalf("fourth header = %q, want right.id", got)
	}
	if got := joined.Cell(0, 5); got != "Tokyo" {
		t.Fatalf("first joined city = %q, want Tokyo", got)
	}
	if got := joined.Cell(1, 5); got != "Kyoto" {
		t.Fatalf("second joined city = %q, want Kyoto", got)
	}
	if got := joined.Cell(2, 5); got != "" {
		t.Fatalf("unmatched joined city = %q, want empty", got)
	}
	if got := joined.Columns[2].Kind; got != KindInt {
		t.Fatalf("left score kind = %s, want int", got)
	}
}

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

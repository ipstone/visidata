package sheet

import "testing"

func TestAddExprColumnWithExplicitName(t *testing.T) {
	sh := New("orders.csv", "/tmp/orders.csv", []string{"qty", "price", "city"})
	sh.AddRow([]string{"2", "1.5", "Tokyo"})
	sh.AddRow([]string{"4", "2.0", "Osaka"})
	sh.InferColumnKinds()

	name, err := sh.AddExprColumn("total := qty * price")
	if err != nil {
		t.Fatalf("AddExprColumn returned error: %v", err)
	}
	if name != "total" {
		t.Fatalf("expr column name = %q, want total", name)
	}
	if got := len(sh.Columns); got != 4 {
		t.Fatalf("len(columns) = %d, want 4", got)
	}
	if got := sh.Cell(0, 3); got != "3" {
		t.Fatalf("first expr value = %q, want 3", got)
	}
	if got := sh.Cell(1, 3); got != "8" {
		t.Fatalf("second expr value = %q, want 8", got)
	}
	if got := sh.Columns[3].Kind; got != KindInt {
		t.Fatalf("expr kind = %s, want int", got)
	}
	if got := sh.CursorCol; got != 3 {
		t.Fatalf("cursor col = %d, want 3", got)
	}
}

func TestAddExprColumnSupportsAliasesAndAutoName(t *testing.T) {
	sh := New("orders.csv", "/tmp/orders.csv", []string{"Order Qty", "City"})
	sh.AddRow([]string{"2", "Tokyo"})
	sh.AddRow([]string{"4", "Osaka"})
	sh.InferColumnKinds()

	name, err := sh.AddExprColumn(`str(col1 * 10) + "-" + City`)
	if err != nil {
		t.Fatalf("AddExprColumn returned error: %v", err)
	}
	if name != "expr_1" {
		t.Fatalf("auto expr column name = %q, want expr_1", name)
	}
	if got := sh.Cell(0, 2); got != "20-Tokyo" {
		t.Fatalf("first expr value = %q, want 20-Tokyo", got)
	}
	if got := sh.Columns[2].Kind; got != KindString {
		t.Fatalf("expr kind = %s, want string", got)
	}
}

func TestAddExprColumnRejectsEmptyExpression(t *testing.T) {
	sh := New("orders.csv", "/tmp/orders.csv", []string{"qty"})
	if _, err := sh.AddExprColumn("   "); err == nil {
		t.Fatal("expected empty expression to fail")
	}
}

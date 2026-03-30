package sheet

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportCSV(t *testing.T) {
	sh := New("people.csv", "/tmp/people.csv", []string{"name", "note"})
	sh.AddRow([]string{"Alice", "hello, world"})
	sh.AddRow([]string{"Bob", "multi\nline"})

	path := filepath.Join(t.TempDir(), "people.csv")
	if err := sh.Export(path); err != nil {
		t.Fatalf("Export returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	got := string(data)
	for _, want := range []string{
		"name,note",
		`Alice,"hello, world"`,
		"Bob,\"multi\nline\"",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("export missing %q:\n%s", want, got)
		}
	}
}

func TestExportJSONPreservesTypedValues(t *testing.T) {
	sh := New("people.json", "/tmp/people.json", []string{"name", "age", "score", "active"})
	sh.AddRow([]string{"Alice", "30", "12.5", "true"})
	sh.Columns[1].Kind = KindInt
	sh.Columns[2].Kind = KindFloat
	sh.Columns[3].Kind = KindBool

	path := filepath.Join(t.TempDir(), "people.json")
	if err := sh.Export(path); err != nil {
		t.Fatalf("Export returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}

	var got []map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal returned error: %v\n%s", err, string(data))
	}
	if age, ok := got[0]["age"].(float64); !ok || age != 30 {
		t.Fatalf("age = %#v, want numeric 30", got[0]["age"])
	}
	if score, ok := got[0]["score"].(float64); !ok || score != 12.5 {
		t.Fatalf("score = %#v, want numeric 12.5", got[0]["score"])
	}
	if active, ok := got[0]["active"].(bool); !ok || !active {
		t.Fatalf("active = %#v, want true", got[0]["active"])
	}
}

func TestSuggestedSavePathUsesSourceFormat(t *testing.T) {
	tsv := New("people.tsv", "/tmp/people.tsv", []string{"name"})
	if got := tsv.SuggestedSavePath(); got != "/tmp/people.vdgo.tsv" {
		t.Fatalf("SuggestedSavePath(tsv) = %q, want %q", got, "/tmp/people.vdgo.tsv")
	}

	stdin := New("-", "-", []string{"name"})
	if got := stdin.SuggestedSavePath(); got != "stdin.vdgo.csv" {
		t.Fatalf("SuggestedSavePath(stdin) = %q, want %q", got, "stdin.vdgo.csv")
	}
}

func TestExportSkipsHiddenColumnsAndUsesOverrides(t *testing.T) {
	sh := New("people.csv", "/tmp/people.csv", []string{"name", "city", "amount"})
	sh.AddRow([]string{"Alice", "Tokyo", "$30"})
	sh.AddRow([]string{"Bob", "Osaka", "$20"})
	sh.Columns[2].OverrideKind = KindCurrency
	sh.Columns[1].Hidden = true

	path := filepath.Join(t.TempDir(), "people.json")
	if err := sh.Export(path); err != nil {
		t.Fatalf("Export returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	var got []map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal returned error: %v\n%s", err, string(data))
	}
	if _, exists := got[0]["city"]; exists {
		t.Fatalf("hidden column should not be exported: %#v", got[0])
	}
	if amount, ok := got[0]["amount"].(float64); !ok || amount != 30 {
		t.Fatalf("amount = %#v, want numeric 30", got[0]["amount"])
	}
}

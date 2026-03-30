package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadJSONL(t *testing.T) {
	sh, err := Load(samplePath("benchmark.jsonl"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(sh.Rows) == 0 {
		t.Fatal("expected jsonl fixture rows to load")
	}
	if len(sh.Columns) != 7 {
		t.Fatalf("len(columns) = %d, want 7", len(sh.Columns))
	}
	if sh.Columns[0].Name != "Customer" {
		t.Fatalf("first sorted column = %q, want %q", sh.Columns[0].Name, "Customer")
	}
	foundQuantity := false
	for _, col := range sh.Columns {
		if col.Name == "Quantity" {
			foundQuantity = true
			if col.Kind != "int" {
				t.Fatalf("Quantity kind = %q, want %q", col.Kind, "int")
			}
		}
	}
	if !foundQuantity {
		t.Fatal("expected Quantity column to be present")
	}
}

func TestLoadJSONDocument(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "items.json")
	content := []byte(`# comment
[
  {"id": 1, "name": "alpha"},
  {"id": 2, "name": "beta"}
]
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	sh, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(sh.Rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(sh.Rows))
	}
	if len(sh.Columns) != 2 {
		t.Fatalf("len(columns) = %d, want 2", len(sh.Columns))
	}
}

func TestLoadJSONObjectAsSingleRow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "item.json")
	if err := os.WriteFile(path, []byte("{\"id\": 3, \"name\": \"gamma\"}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	sh, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(sh.Rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(sh.Rows))
	}
}

func TestLoadJSONDocumentWithOnlyCommentsFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "comments.json")
	if err := os.WriteFile(path, []byte("# comment\n// another comment\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatal("expected comments-only JSON document to return an error")
	}
}

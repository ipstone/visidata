package loader

import (
	"path/filepath"
	"testing"
)

func samplePath(name string) string {
	return filepath.Join("..", "..", "..", "sample_data", name)
}

func repoPath(name string) string {
	return filepath.Join("..", "..", "..", name)
}

func TestLoadTSV(t *testing.T) {
	sh, err := Load(samplePath("a.tsv"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(sh.Columns) != 2 {
		t.Fatalf("len(columns) = %d, want 2", len(sh.Columns))
	}
	if len(sh.Rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(sh.Rows))
	}
	if got := sh.Columns[0].Name; got != "key" {
		t.Fatalf("first column = %q, want %q", got, "key")
	}
}

func TestLoadCSVAndInferKinds(t *testing.T) {
	sh, err := Load(samplePath("benchmark.csv"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(sh.Columns) != 7 {
		t.Fatalf("len(columns) = %d, want 7", len(sh.Columns))
	}
	if len(sh.Rows) == 0 {
		t.Fatal("expected rows to be loaded")
	}
	if sh.Columns[4].Kind != "int" {
		t.Fatalf("quantity kind = %q, want %q", sh.Columns[4].Kind, "int")
	}
}

func TestLoadAllowsMalformedCSVRows(t *testing.T) {
	sh, err := Load(samplePath("errors.csv"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(sh.Rows) == 0 {
		t.Fatal("expected malformed fixture rows to load")
	}
}

func TestLoadSQLiteDefaultsToFirstTable(t *testing.T) {
	sh, err := Load(repoPath(filepath.Join("tests", "without_rowid.db")))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got := sh.Name; got != "without_rowid.db:withoutrowid" {
		t.Fatalf("sheet name = %q, want %q", got, "without_rowid.db:withoutrowid")
	}
	if len(sh.Rows) != 2 || sh.Cell(0, 1) != "abc" {
		t.Fatalf("unexpected sqlite rows: %#v", sh.Rows)
	}
}

func TestLoadSQLiteSpecificTable(t *testing.T) {
	sh, err := LoadWithOptions(repoPath(filepath.Join("tests", "without_rowid.db")), Options{Table: "withrowid"})
	if err != nil {
		t.Fatalf("LoadWithOptions returned error: %v", err)
	}
	if got := sh.Name; got != "without_rowid.db:withrowid" {
		t.Fatalf("sheet name = %q, want %q", got, "without_rowid.db:withrowid")
	}
	if got := sh.Columns[0].Kind; got != "int" {
		t.Fatalf("id kind = %q, want int", got)
	}
}

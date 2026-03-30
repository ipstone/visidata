package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFixedWithHeader(t *testing.T) {
	sh, err := Load(samplePath("test.fixed"))
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(sh.Columns) != 3 {
		t.Fatalf("len(columns) = %d, want 3", len(sh.Columns))
	}
	if got := sh.Columns[0].Name; got != "a" {
		t.Fatalf("first column = %q, want %q", got, "a")
	}
	if len(sh.Rows) != 3 {
		t.Fatalf("len(rows) = %d, want 3", len(sh.Rows))
	}
	if got := sh.Cell(0, 0); got != "d  " {
		t.Fatalf("first data cell = %q, want %q", got, "d  ")
	}
}

func TestLoadFixedWithoutHeaderPreservesLeadingSpaces(t *testing.T) {
	sh, err := LoadWithOptions(samplePath("test-fixed-leadingspaces.txt"), Options{
		Format: "fixed",
		Header: 0,
	})
	if err != nil {
		t.Fatalf("LoadWithOptions returned error: %v", err)
	}

	if len(sh.Columns) != 3 {
		t.Fatalf("len(columns) = %d, want 3", len(sh.Columns))
	}
	if got := sh.Columns[0].Name; got != "" {
		t.Fatalf("first column name = %q, want empty", got)
	}
	if got := sh.Cell(0, 0); got != "  1" {
		t.Fatalf("first data cell = %q, want %q", got, "  1")
	}
}

func TestLoadDirectoryListsVisibleEntries(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "alpha.csv"), []byte("a,b\n1,2\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".hidden.txt"), []byte("secret\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatalf("Mkdir returned error: %v", err)
	}

	sh, err := Load(root)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if len(sh.Rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(sh.Rows))
	}
	if got := sh.Cell(0, 1); got != "alpha.csv" {
		t.Fatalf("first filename = %q, want alpha.csv", got)
	}
	if got := sh.Cell(1, 2); got != "/" {
		t.Fatalf("directory ext marker = %q, want /", got)
	}
	if got := sh.Columns[3].Kind; got != "int" {
		t.Fatalf("size kind = %q, want int", got)
	}
}

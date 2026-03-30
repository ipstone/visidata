package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestVDGOPreviewsFixture(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/vdgo", "-n", "2", "../sample_data/a.tsv")
	cmd.Dir = ".."

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run returned error: %v\n%s", err, string(out))
	}

	output := string(out)
	for _, want := range []string{
		"a.tsv: 2 row(s) x 2 column(s)",
		"key <string>",
		"source_sheet <string>",
		"a_only",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestVDGOPreviewsJSONLFixture(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/vdgo", "-n", "2", "../sample_data/benchmark.jsonl")
	cmd.Dir = ".."

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run returned error: %v\n%s", err, string(out))
	}

	output := string(out)
	for _, want := range []string{
		"benchmark.jsonl:",
		"Customer <string>",
		"Quantity <int>",
		"Robert Armstrong",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestVDGOPreviewsJSONFromStdin(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/vdgo", "-n", "2", "-")
	cmd.Dir = ".."
	cmd.Stdin = strings.NewReader("{\"name\":\"Alice\",\"age\":30}\n{\"name\":\"Bob\",\"age\":40}\n")

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run returned error: %v\n%s", err, string(out))
	}

	output := string(out)
	for _, want := range []string{
		"stdin:",
		"age <int>",
		"name <string>",
		"Alice",
		"Bob",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestVDGOPreviewsSQLiteFixture(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/vdgo", "-n", "2", "../tests/without_rowid.db")
	cmd.Dir = ".."

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run returned error: %v\n%s", err, string(out))
	}

	output := string(out)
	for _, want := range []string{
		"without_rowid.db:withoutrowid: 2 row(s) x 2 column(s)",
		"id <int>",
		"datum <string>",
		"abc",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestVDGOPreviewsSpecificSQLiteTable(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/vdgo", "-n", "2", "-table", "withrowid", "../tests/without_rowid.db")
	cmd.Dir = ".."

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run returned error: %v\n%s", err, string(out))
	}

	output := string(out)
	if !strings.Contains(output, "without_rowid.db:withrowid") {
		t.Fatalf("output missing specific sqlite table name:\n%s", output)
	}
}

func TestVDGOPreviewsFixedWidthFixture(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/vdgo", "-n", "2", "../sample_data/test.fixed")
	cmd.Dir = ".."

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run returned error: %v\n%s", err, string(out))
	}

	output := string(out)
	for _, want := range []string{
		"test.fixed: 3 row(s) x 3 column(s)",
		"a <string>",
		"b <string>",
		"d          |  e",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestVDGOPreviewsFixedWidthWithoutHeader(t *testing.T) {
	cmd := exec.Command("go", "run", "./cmd/vdgo", "-n", "2", "-filetype", "fixed", "-header", "0", "../sample_data/test-fixed-leadingspaces.txt")
	cmd.Dir = ".."

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run returned error: %v\n%s", err, string(out))
	}

	output := string(out)
	for _, want := range []string{
		"test-fixed-leadingspaces.txt: 3 row(s) x 3 column(s)",
		"  1",
		"aaa",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestVDGOPreviewsDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "alpha.csv"), []byte("a,b\n1,2\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "nested"), 0o755); err != nil {
		t.Fatalf("Mkdir returned error: %v", err)
	}

	cmd := exec.Command("go", "run", "./cmd/vdgo", "-n", "5", root)
	cmd.Dir = ".."

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run returned error: %v\n%s", err, string(out))
	}

	output := string(out)
	for _, want := range []string{
		"2 row(s) x 5 column(s)",
		"directory <string>",
		"filename <string>",
		"alpha.csv",
		"nested",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

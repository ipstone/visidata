package loader

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestLoadYAMLTopLevelSequence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "people.yaml")
	content := []byte("people:\n  - id: 1\n    name: Alice\n  - id: 2\n    name: Bob\n")
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
	if got := sh.Cell(1, 1); got != "Bob" {
		t.Fatalf("second name = %q, want Bob", got)
	}
}

func TestLoadTOMLArrayOfTables(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "people.toml")
	content := []byte("[[people]]\nid = 1\nname = \"Alice\"\n\n[[people]]\nid = 2\nname = \"Bob\"\n")
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
	if got := sh.Columns[0].Name; got != "id" {
		t.Fatalf("first column = %q, want id", got)
	}
}

func TestLoadXMLRepeatedElements(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "people.xml")
	content := []byte("<people><person><id>1</id><name>Alice</name></person><person><id>2</id><name>Bob</name></person></people>")
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
	if got := sh.Cell(0, 1); got != "Alice" {
		t.Fatalf("first name = %q, want Alice", got)
	}
}

func TestLoadMarkdownTable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "people.md")
	content := []byte("# People\n\n| id | name |\n| -- | ---- |\n| 1 | Alice |\n| 2 | Bob |\n")
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
	if got := sh.Cell(1, 1); got != "Bob" {
		t.Fatalf("second name = %q, want Bob", got)
	}
}

func TestLoadHTMLTable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "people.html")
	content := []byte("<html><body><table><tr><th>id</th><th>name</th></tr><tr><td>1</td><td>Alice</td></tr><tr><td>2</td><td>Bob</td></tr></table></body></html>")
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
	if got := sh.Cell(0, 1); got != "Alice" {
		t.Fatalf("first name = %q, want Alice", got)
	}
}

func TestLoadCompressedCSV(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "people.csv.gz")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	gz := gzip.NewWriter(file)
	if _, err := io.WriteString(gz, "id,name\n1,Alice\n2,Bob\n"); err != nil {
		t.Fatalf("WriteString returned error: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	sh, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(sh.Rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(sh.Rows))
	}
	if got := sh.Cell(1, 1); got != "Bob" {
		t.Fatalf("second name = %q, want Bob", got)
	}
}

func TestLoadHTTPURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "id,name\n1,Alice\n2,Bob\n")
	}))
	defer server.Close()

	sh, err := Load(server.URL + "/people.csv")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(sh.Rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(sh.Rows))
	}
	if got := sh.Cell(0, 1); got != "Alice" {
		t.Fatalf("first name = %q, want Alice", got)
	}
}

func TestLoadZipArchive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.zip")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	zw := zip.NewWriter(file)
	w, err := zw.Create("alpha.csv")
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if _, err := io.WriteString(w, "a,b\n1,2\n"); err != nil {
		t.Fatalf("WriteString returned error: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	sh, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(sh.Rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(sh.Rows))
	}
	if got := sh.Cell(0, 0); got != "alpha.csv" {
		t.Fatalf("archive entry path = %q, want alpha.csv", got)
	}
}

func TestLoadTarGzArchive(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bundle.tar.gz")
	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	content := []byte("hello\n")
	header := &tar.Header{Name: "notes.txt", Mode: 0o644, Size: int64(len(content))}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatalf("WriteHeader returned error: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	sh, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(sh.Rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(sh.Rows))
	}
	if got := sh.Cell(0, 0); got != "notes.txt" {
		t.Fatalf("archive entry path = %q, want notes.txt", got)
	}
}

func TestLoadXLSXDefaultAndSelectedSheet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "people.xlsx")

	workbook := excelize.NewFile()
	defer workbook.Close()
	defaultSheet := workbook.GetSheetName(0)
	if err := workbook.SetSheetName(defaultSheet, "People"); err != nil {
		t.Fatalf("SetSheetName returned error: %v", err)
	}
	if err := workbook.SetCellValue("People", "A1", "id"); err != nil {
		t.Fatalf("SetCellValue returned error: %v", err)
	}
	if err := workbook.SetCellValue("People", "B1", "name"); err != nil {
		t.Fatalf("SetCellValue returned error: %v", err)
	}
	if err := workbook.SetCellValue("People", "A2", 1); err != nil {
		t.Fatalf("SetCellValue returned error: %v", err)
	}
	if err := workbook.SetCellValue("People", "B2", "Alice"); err != nil {
		t.Fatalf("SetCellValue returned error: %v", err)
	}
	if _, err := workbook.NewSheet("Cities"); err != nil {
		t.Fatalf("NewSheet returned error: %v", err)
	}
	if err := workbook.SetCellValue("Cities", "A1", "city"); err != nil {
		t.Fatalf("SetCellValue returned error: %v", err)
	}
	if err := workbook.SetCellValue("Cities", "A2", "Tokyo"); err != nil {
		t.Fatalf("SetCellValue returned error: %v", err)
	}
	if err := workbook.SaveAs(path); err != nil {
		t.Fatalf("SaveAs returned error: %v", err)
	}

	sh, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got := sh.Name; got != "people.xlsx:People" {
		t.Fatalf("sheet name = %q, want people.xlsx:People", got)
	}
	if len(sh.Rows) != 1 || sh.Cell(0, 1) != "Alice" {
		t.Fatalf("unexpected xlsx data: %#v", sh.Rows)
	}
	if got := sh.Columns[0].Kind; got != "int" {
		t.Fatalf("id kind = %q, want int", got)
	}

	selected, err := LoadWithOptions(path, Options{Table: "Cities"})
	if err != nil {
		t.Fatalf("LoadWithOptions returned error: %v", err)
	}
	if got := selected.Name; got != "people.xlsx:Cities" {
		t.Fatalf("sheet name = %q, want people.xlsx:Cities", got)
	}
	if len(selected.Rows) != 1 || selected.Cell(0, 0) != "Tokyo" {
		t.Fatalf("unexpected selected xlsx data: %#v", selected.Rows)
	}
}

func TestStripCompressionSuffix(t *testing.T) {
	tests := map[string]string{
		"people.csv.gz":     "people.csv",
		"bundle.tar.gz":     "bundle.tar",
		"https://x/a.tgz":   "https://x/a.tar",
		"https://x/a.csv?x": "https://x/a.csv?x",
		"bundle.tar.zst":    "bundle.tar",
		"bundle.data":       "bundle.data",
	}

	for input, want := range tests {
		if got := stripCompressionSuffix(input); got != want {
			t.Fatalf("stripCompressionSuffix(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNormalizedExtHandlesCompressedSuffixes(t *testing.T) {
	tests := map[string]string{
		"people.csv.gz": ".csv",
		"bundle.tar.gz": ".tar",
		"bundle.zip":    ".zip",
		"people.xlsx":   ".xlsx",
	}
	for input, want := range tests {
		if got := normalizedExt(input); got != want {
			t.Fatalf("normalizedExt(%q) = %q, want %q", input, got, want)
		}
	}
}

package display

import (
	"strings"
	"testing"

	"github.com/ipstone/visidata/go-visidata/pkg/sheet"
)

func TestRenderPreview(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.AddRow([]string{"Bob", "40"})
	sh.InferColumnKinds()

	out := RenderPreview(sh, 1)
	wants := []string{
		"people.csv: 2 row(s) x 2 column(s)",
		"name <string>",
		"age <int>",
		"Alice",
		"... showing first 1 of 2 row(s)",
	}
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Fatalf("preview missing %q in output:\n%s", want, out)
		}
	}
}

func TestRenderPreviewAccountsForWideCharacters(t *testing.T) {
	sh := sheet.New("pets.csv", "/tmp/pets.csv", []string{"name", "city"})
	sh.AddRow([]string{"桜 高橋", "Tokyo"})
	sh.AddRow([]string{"Alice", "Osaka"})
	sh.InferColumnKinds()

	out := RenderPreview(sh, 2)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 5 {
		t.Fatalf("expected at least 5 output lines, got %d: %q", len(lines), out)
	}

	for _, line := range lines[3:5] {
		if !strings.Contains(line, " | ") {
			t.Fatalf("expected aligned column separator in %q", line)
		}
	}
}

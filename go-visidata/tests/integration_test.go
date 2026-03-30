package tests

import (
	"os/exec"
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

package sheet

import "testing"

func TestCopyCell(t *testing.T) {
	sh := New("test.csv", "/tmp/test.csv", []string{"name", "city"})
	sh.AddRow([]string{"Alice", "Tokyo"})

	if ok := sh.CopyCell(0, 1); !ok {
		t.Fatal("CopyCell returned false, want true")
	}
	if got := sh.Clipboard.Content; got != "Tokyo" {
		t.Fatalf("clipboard content = %q, want %q", got, "Tokyo")
	}
	if got := sh.Clipboard.Kind; got != "cell" {
		t.Fatalf("clipboard kind = %q, want cell", got)
	}
}

func TestCopySelectedRowsOrCurrentUsesSelection(t *testing.T) {
	sh := New("test.csv", "/tmp/test.csv", []string{"name", "city"})
	sh.AddRow([]string{"Alice", "Tokyo"})
	sh.AddRow([]string{"Bob", "Osaka"})
	sh.ToggleSelected(0)
	sh.ToggleSelected(1)

	if ok := sh.CopySelectedRowsOrCurrent(); !ok {
		t.Fatal("CopySelectedRowsOrCurrent returned false, want true")
	}
	if got := sh.Clipboard.Content; got != "Alice\tTokyo\nBob\tOsaka" {
		t.Fatalf("clipboard content = %q, want selected rows TSV", got)
	}
	if got := sh.Clipboard.Kind; got != "selected rows" {
		t.Fatalf("clipboard kind = %q, want selected rows", got)
	}
	if got := sh.Clipboard.Count; got != 2 {
		t.Fatalf("clipboard count = %d, want 2", got)
	}
}

func TestCopySelectedRowsOrCurrentFallsBackToCurrentRow(t *testing.T) {
	sh := New("test.csv", "/tmp/test.csv", []string{"name", "city"})
	sh.AddRow([]string{"Alice", "Tokyo"})
	sh.AddRow([]string{"Bob", "Osaka"})
	sh.SetCursorRow(1)

	if ok := sh.CopySelectedRowsOrCurrent(); !ok {
		t.Fatal("CopySelectedRowsOrCurrent returned false, want true")
	}
	if got := sh.Clipboard.Content; got != "Bob\tOsaka" {
		t.Fatalf("clipboard content = %q, want current row TSV", got)
	}
	if got := sh.Clipboard.Kind; got != "row" {
		t.Fatalf("clipboard kind = %q, want row", got)
	}
}

func TestClipboardPreviewNormalizesTabsAndNewlines(t *testing.T) {
	sh := New("test.csv", "/tmp/test.csv", []string{"a"})
	sh.Clipboard = Clipboard{Content: "a\tb\nc", Kind: "selected rows", Count: 2}

	if got := sh.ClipboardPreview(0); got != "a ⇥ b ⏎ c" {
		t.Fatalf("ClipboardPreview = %q, want %q", got, "a ⇥ b ⏎ c")
	}
	if got := sh.ClipboardPreview(5); got != "a ⇥ b…" {
		t.Fatalf("ClipboardPreview truncation = %q, want %q", got, "a ⇥ b…")
	}
}

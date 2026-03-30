package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/ipstone/visidata/go-visidata/pkg/sheet"
)

func TestHandleKeyMovesCursorAndViewport(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age", "city"})
	for _, row := range [][]string{
		{"Alice", "30", "Tokyo"},
		{"Bob", "40", "Osaka"},
		{"Carol", "50", "Kyoto"},
		{"Dave", "60", "Nagoya"},
	} {
		sh.AddRow(row)
	}
	sh.InferColumnKinds()

	app := New(sh, nil)
	app.ensureVisible(20, 6)

	app.HandleKey(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
	app.ensureVisible(20, 6)

	if sh.CursorRow != 2 || sh.CursorCol != 2 {
		t.Fatalf("cursor = (%d,%d), want (2,2)", sh.CursorRow, sh.CursorCol)
	}
	if app.RowOffset == 0 {
		t.Fatalf("expected row viewport to scroll, got offset %d", app.RowOffset)
	}
	if app.ColOffset == 0 {
		t.Fatalf("expected col viewport to scroll, got offset %d", app.ColOffset)
	}
}

func TestDrawRendersStatusAndData(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.AddRow([]string{"Bob", "40"})
	sh.InferColumnKinds()

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(60, 8)

	app := New(sh, screen)
	app.Draw()

	lines := snapshot(screen, 60, 8)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		"people.csv: 2 row(s) x 2 column(s)",
		"name <string>",
		"Alice",
		"row 1/2",
		"q quit",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("screen missing %q:\n%s", want, joined)
		}
	}
}

func TestHandleKeySortSearchAndSelection(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age", "city"})
	for _, row := range [][]string{
		{"Alice", "30", "Tokyo"},
		{"Bob", "20", "Osaka"},
		{"Carol", "10", "Kyoto"},
	} {
		sh.AddRow(row)
	}
	sh.InferColumnKinds()

	app := New(sh, nil)
	sh.SetCursorCol(1)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '[', tcell.ModNone))
	if got := sh.Cell(0, 0); got != "Carol" {
		t.Fatalf("first row after sort = %q, want Carol", got)
	}

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 's', tcell.ModNone))
	if got := sh.SelectedCount(); got != 1 {
		t.Fatalf("SelectedCount = %d, want 1", got)
	}

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'c', tcell.ModNone))
	if got := sh.Clipboard.Content; got != "10" {
		t.Fatalf("clipboard after c = %q, want %q", got, "10")
	}

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'C', tcell.ModNone))
	if got := sh.Clipboard.Content; got != "Carol\t10\tKyoto" {
		t.Fatalf("clipboard after C = %q, want current selected row", got)
	}

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '/', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'o', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if sh.SearchState.Query != "o" {
		t.Fatalf("search query = %q, want %q", sh.SearchState.Query, "o")
	}
	if len(sh.SearchState.Matches) == 0 {
		t.Fatal("expected search matches")
	}

	current := sh.SearchState.CurrentMatch
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'n', tcell.ModNone))
	if sh.SearchState.CurrentMatch == current {
		t.Fatal("expected n to advance to the next match")
	}

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'd', tcell.ModNone))
	if len(sh.Rows) != 2 {
		t.Fatalf("row count after delete = %d, want 2", len(sh.Rows))
	}
	if got := sh.Clipboard.Kind; got != "deleted rows" {
		t.Fatalf("clipboard kind after delete = %q, want deleted rows", got)
	}
}

func TestHandleKeyColumnOpsAndRegexSelection(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age", "city"})
	sh.AddRow([]string{"Alice", "30", "Tokyo"})
	sh.AddRow([]string{"Bob", "20", "Osaka"})
	sh.AddRow([]string{"Carol", "10", "Kyoto"})
	sh.InferColumnKinds()

	app := New(sh, nil)

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '^', tcell.ModNone))
	for _, r := range "person" {
		app.HandleKey(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if got := sh.Columns[0].Name; got != "person" {
		t.Fatalf("renamed column = %q, want person", got)
	}

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '_', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '1', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '2', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if got := sh.Columns[0].Width; got != 12 {
		t.Fatalf("column width = %d, want 12", got)
	}

	sh.SetCursorCol(1)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '$', tcell.ModNone))
	if got := sh.Columns[1].EffectiveKind(); got != sheet.KindCurrency {
		t.Fatalf("effective kind = %s, want currency", got)
	}

	sh.SetCursorCol(2)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '|', tcell.ModNone))
	for _, r := range "o$" {
		app.HandleKey(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if got := sh.SelectedCount(); got != 2 {
		t.Fatalf("SelectedCount after regex select = %d, want 2", got)
	}

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '\\', tcell.ModNone))
	for _, r := range "Kyoto" {
		app.HandleKey(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if got := sh.SelectedCount(); got != 1 {
		t.Fatalf("SelectedCount after regex unselect = %d, want 1", got)
	}

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '-', tcell.ModNone))
	if got := sh.HiddenColumnCount(); got != 1 {
		t.Fatalf("HiddenColumnCount = %d, want 1", got)
	}

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'H', tcell.ModNone))
	if got := sh.HiddenColumnCount(); got != 0 {
		t.Fatalf("HiddenColumnCount after H = %d, want 0", got)
	}
}

func TestDrawRendersSortSearchAndSelectionStatus(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.AddRow([]string{"Bob", "20"})
	sh.InferColumnKinds()
	sh.ToggleSelected(1)
	sh.ToggleSort(1, sheet.SortAsc)
	sh.Search("o")
	sh.CopyCell(0, 1)

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(220, 8)

	app := New(sh, screen)
	app.Draw()

	lines := snapshot(screen, 220, 8)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		"* ",
		"sort age↑",
		"search \"o\"",
		"sel 1",
		"cell \"20\"",
		"c/C copy",
		"d delete",
		"S save",
		"- hide",
		"^ rename",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("screen missing %q:\n%s", want, joined)
		}
	}
}

func TestDrawRendersCopiedRowsStatus(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.AddRow([]string{"Bob", "20"})
	sh.InferColumnKinds()
	sh.ToggleSelected(0)
	sh.ToggleSelected(1)
	sh.CopySelectedRowsOrCurrent()

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(100, 8)

	app := New(sh, screen)
	app.Draw()

	lines := snapshot(screen, 100, 8)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		"selected rows 2",
		`"Alice ⇥ 30 ⏎ Bob ⇥ 20"`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("screen missing %q:\n%s", want, joined)
		}
	}
}

func TestHandleKeySaveSuggestedWritesFile(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "people.tsv")
	sh := sheet.New(source, source, []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.AddRow([]string{"Bob", "20"})

	app := New(sh, nil)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'S', tcell.ModNone))

	savePath := filepath.Join(root, "people.vdgo.tsv")
	data, err := os.ReadFile(savePath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(data), "name\tage") || !strings.Contains(string(data), "Alice\t30") {
		t.Fatalf("saved file missing expected TSV content:\n%s", string(data))
	}
	if !strings.Contains(sh.Status, "saved ") {
		t.Fatalf("status = %q, want saved message", sh.Status)
	}
}

func TestDrawRendersDeleteAndSaveStatus(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "people.csv")
	sh := sheet.New(source, source, []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.AddRow([]string{"Bob", "20"})
	sh.ToggleSelected(0)
	sh.DeleteSelectedRowsOrCurrent()
	sh.Status = "deleted 1 row(s)"

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(160, 8)

	app := New(sh, screen)
	app.Draw()

	lines := snapshot(screen, 160, 8)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		"deleted rows 1",
		"deleted 1 row(s)",
		"d delete",
		"S save",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("screen missing %q:\n%s", want, joined)
		}
	}
}

func TestDrawRendersInputPromptAndHiddenColumnsStatus(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age", "city"})
	sh.AddRow([]string{"Alice", "30", "Tokyo"})
	sh.AddRow([]string{"Bob", "20", "Osaka"})
	sh.InferColumnKinds()
	if err := sh.ToggleHidden(2); err != nil {
		t.Fatalf("ToggleHidden returned error: %v", err)
	}

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(180, 8)

	app := New(sh, screen)
	app.beginInput(inputModeRename, "name")
	app.Draw()

	lines := snapshot(screen, 180, 8)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		"2/3 column(s)",
		"Rename column: name",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("screen missing %q:\n%s", want, joined)
		}
	}
}

func snapshot(screen tcell.SimulationScreen, width, height int) []string {
	lines := make([]string, 0, height)
	for y := 0; y < height; y++ {
		var b strings.Builder
		for x := 0; x < width; x++ {
			mainc, combc, _, _ := screen.GetContent(x, y)
			if len(combc) > 0 {
				b.WriteRune(combc[0])
				continue
			}
			if mainc == 0 {
				mainc = ' '
			}
			b.WriteRune(mainc)
		}
		lines = append(lines, strings.TrimRight(b.String(), " "))
	}
	return lines
}

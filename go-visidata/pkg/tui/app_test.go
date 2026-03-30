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
	screen.SetSize(560, 8)

	app := New(sh, screen)
	app.Draw()

	lines := snapshot(screen, 560, 8)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		"* ",
		"sort age↑",
		"search \"o\"",
		"sel 1",
		"cell \"20\"",
		"f freeze",
		"D dedupe",
		"c / C copy",
		"d delete",
		"S save",
		"- / H columns-visibility",
		"^ / _ columns-edit",
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

func TestHandleKeyOpensDerivedSheetsAndPopsBack(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"city", "age"})
	sh.AddRow([]string{"Tokyo", "30"})
	sh.AddRow([]string{"Tokyo", "20"})
	sh.AddRow([]string{"Osaka", "10"})
	sh.InferColumnKinds()

	app := New(sh, nil)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'f', tcell.ModNone))
	if got := app.Sheet.Name; got != "freeze:people.csv" {
		t.Fatalf("freeze sheet name = %q, want freeze:people.csv", got)
	}
	if quit := app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone)); quit {
		t.Fatal("q should pop frozen sheet before quitting")
	}

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'D', tcell.ModNone))
	if got := app.Sheet.Name; got != "dedupe:people.csv:city" {
		t.Fatalf("dedupe sheet name = %q, want dedupe:people.csv:city", got)
	}
	if quit := app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone)); quit {
		t.Fatal("q should pop dedupe sheet before quitting")
	}

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'F', tcell.ModNone))
	if got := app.Sheet.Name; got != "freq:people.csv:city" {
		t.Fatalf("frequency sheet name = %q, want freq:people.csv:city", got)
	}
	if len(app.stack) != 2 {
		t.Fatalf("stack depth = %d, want 2", len(app.stack))
	}
	if quit := app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone)); quit {
		t.Fatal("q should pop derived sheet before quitting")
	}
	if got := app.Sheet.Name; got != "people.csv" {
		t.Fatalf("sheet after pop = %q, want people.csv", got)
	}

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'I', tcell.ModNone))
	if got := app.Sheet.Name; got != "describe:people.csv" {
		t.Fatalf("describe sheet name = %q, want describe:people.csv", got)
	}
	_ = app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone))

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'T', tcell.ModNone))
	if got := app.Sheet.Name; got != "transpose:people.csv" {
		t.Fatalf("transpose sheet name = %q, want transpose:people.csv", got)
	}
	if quit := app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone)); quit {
		t.Fatal("q should return to the root sheet")
	}
	if quit := app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone)); !quit {
		t.Fatal("q on the root sheet should quit")
	}
}

func TestHandleKeyOpensMetaSheetsAndSelectsStackEntry(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"city", "age"})
	sh.AddRow([]string{"Tokyo", "30"})
	sh.AddRow([]string{"Osaka", "20"})
	sh.InferColumnKinds()

	app := New(sh, nil)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'F', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'V', tcell.ModNone))
	if got := app.Sheet.Name; got != "sheets" {
		t.Fatalf("sheets meta name = %q, want sheets", got)
	}
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if got := app.Sheet.Name; got != "people.csv" {
		t.Fatalf("sheet after enter = %q, want people.csv", got)
	}
	if len(app.stack) != 1 {
		t.Fatalf("stack depth after selecting root = %d, want 1", len(app.stack))
	}

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'M', tcell.ModNone))
	if got := app.Sheet.Name; got != "columns:people.csv" {
		t.Fatalf("columns meta name = %q, want columns:people.csv", got)
	}
	_ = app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone))

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'O', tcell.ModNone))
	if got := app.Sheet.Name; got != "options:people.csv" {
		t.Fatalf("options meta name = %q, want options:people.csv", got)
	}
	_ = app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone))

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '?', tcell.ModNone))
	if got := app.Sheet.Name; got != "commands" {
		t.Fatalf("commands meta name = %q, want commands", got)
	}
	if got := app.Sheet.Cell(0, 1); got != "quit" {
		t.Fatalf("first command name = %q, want quit", got)
	}
}

func TestHandleKeyEditsColumnsMetasheetSource(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age", "city"})
	sh.AddRow([]string{"Alice", "30", "Tokyo"})
	sh.AddRow([]string{"Bob", "20", "Osaka"})
	sh.InferColumnKinds()

	app := New(sh, nil)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'M', tcell.ModNone))
	app.Sheet.SetCursorRow(1)

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '$', tcell.ModNone))
	if got := sh.Columns[1].EffectiveKind(); got != sheet.KindCurrency {
		t.Fatalf("source type = %s, want currency", got)
	}
	if got := app.Sheet.Cell(1, 2); got != "currency" {
		t.Fatalf("meta type = %q, want currency", got)
	}

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '^', tcell.ModNone))
	for _, r := range "years" {
		app.HandleKey(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if got := sh.Columns[1].Name; got != "years" {
		t.Fatalf("source column name = %q, want years", got)
	}
	if got := app.Sheet.Cell(1, 1); got != "years" {
		t.Fatalf("meta column name = %q, want years", got)
	}

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '_', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '1', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '4', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if got := sh.Columns[1].Width; got != 14 {
		t.Fatalf("source column width = %d, want 14", got)
	}
	if got := app.Sheet.Cell(1, 3); got != "14" {
		t.Fatalf("meta column width = %q, want 14", got)
	}

	app.Sheet.SetCursorRow(2)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '-', tcell.ModNone))
	if got := sh.HiddenColumnCount(); got != 1 {
		t.Fatalf("hidden column count = %d, want 1", got)
	}
	if got := app.Sheet.Cell(2, 4); got != "true" {
		t.Fatalf("meta hidden state = %q, want true", got)
	}

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'H', tcell.ModNone))
	if got := sh.HiddenColumnCount(); got != 0 {
		t.Fatalf("hidden column count after H = %d, want 0", got)
	}
	if got := app.Sheet.Cell(2, 4); got != "false" {
		t.Fatalf("meta hidden state after H = %q, want false", got)
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
	screen.SetSize(560, 8)

	app := New(sh, screen)
	app.Draw()

	lines := snapshot(screen, 560, 8)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		"deleted rows 1",
		"deleted 1 row(s)",
		"f freeze",
		"D dedupe",
		"d delete",
		"F freq",
		"I describe",
		"T transpose",
		"V sheets",
		"M columns",
		"O options",
		"? commands",
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

func TestDrawRendersMetaStackStatus(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.AddRow([]string{"Bob", "20"})

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(220, 8)

	app := New(sh, screen)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'M', tcell.ModNone))
	app.Draw()

	lines := snapshot(screen, 220, 8)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		"columns:people.csv",
		"stack 2",
		"f freeze",
		"V sheets",
		"? commands",
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

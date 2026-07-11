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

func TestHandleKeyEnterOpensRowDetailsSheet(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age", "city"})
	sh.AddRow([]string{"Alice", "30", "Tokyo"})
	sh.AddRow([]string{"Bob", "20", "Osaka"})
	sh.InferColumnKinds()
	sh.SetCursorRow(1)

	app := New(sh, nil)
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if got := app.Sheet.Name; got != "row:people.csv:2" {
		t.Fatalf("row details name = %q, want row:people.csv:2", got)
	}
	if got := app.Sheet.Cell(2, 3); got != "Osaka" {
		t.Fatalf("row details city = %q, want Osaka", got)
	}
	if quit := app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone)); quit {
		t.Fatal("q should pop row details sheet before quitting")
	}
	if got := app.Sheet.Name; got != "people.csv" {
		t.Fatalf("sheet after pop = %q, want people.csv", got)
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
		"age <int>",
		"Alice",
		"30",
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

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'y', tcell.ModNone))
	if got := sh.Clipboard.Content; got != "10" {
		t.Fatalf("clipboard after y = %q, want %q", got, "10")
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

func TestHandleKeyGotoColumnAndRowCommands(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age", "city"})
	sh.AddRow([]string{"Alice", "30", "Tokyo"})
	sh.AddRow([]string{"Bob", "20", "Osaka"})
	sh.AddRow([]string{"Carol", "10", "Kyoto"})
	sh.InferColumnKinds()

	app := New(sh, nil)

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'c', tcell.ModNone))
	for _, r := range "city" {
		app.HandleKey(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if got := sh.CursorCol; got != 2 {
		t.Fatalf("cursor col after c regex = %d, want 2", got)
	}

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'r', tcell.ModNone))
	for _, r := range "Osaka" {
		app.HandleKey(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if got := sh.CursorRow; got != 1 {
		t.Fatalf("cursor row after r regex = %d, want 1", got)
	}

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'z', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'c', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '0', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if got := sh.CursorCol; got != 0 {
		t.Fatalf("cursor col after zc = %d, want 0", got)
	}

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'z', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'r', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '2', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if got := sh.CursorRow; got != 2 {
		t.Fatalf("cursor row after zr = %d, want 2", got)
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

func TestHandleKeyAddsRegexDerivedColumns(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"person", "coord"})
	sh.AddRow([]string{"Alice 30", "10,20"})
	sh.AddRow([]string{"Bob 25", "5,8,13"})

	app := New(sh, nil)

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '{', tcell.ModNone))
	for _, r := range `^(\w+)\s+(\d+)$` {
		app.HandleKey(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if got := len(sh.Columns); got != 4 {
		t.Fatalf("len(columns) after capture = %d, want 4", got)
	}
	if got := sh.Cell(0, 2); got != "Alice" {
		t.Fatalf("capture value = %q, want Alice", got)
	}
	if got := sh.Cell(1, 3); got != "25" {
		t.Fatalf("capture age = %q, want 25", got)
	}

	sh.SetCursorCol(1)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '}', tcell.ModNone))
	for _, r := range `,` {
		app.HandleKey(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if got := len(sh.Columns); got != 7 {
		t.Fatalf("len(columns) after split = %d, want 7", got)
	}
	if got := sh.Cell(0, 6); got != "" {
		t.Fatalf("missing split value = %q, want empty", got)
	}
	if got := sh.Cell(1, 6); got != "13" {
		t.Fatalf("third split value = %q, want 13", got)
	}
}

func TestHandleKeyAddsExpressionColumn(t *testing.T) {
	sh := sheet.New("orders.csv", "/tmp/orders.csv", []string{"qty", "price"})
	sh.AddRow([]string{"2", "1.5"})
	sh.AddRow([]string{"4", "2.0"})
	sh.InferColumnKinds()

	app := New(sh, nil)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '=', tcell.ModNone))
	for _, r := range "total := qty * price" {
		app.HandleKey(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))

	if got := len(sh.Columns); got != 3 {
		t.Fatalf("len(columns) = %d, want 3", got)
	}
	if got := sh.Columns[2].Name; got != "total" {
		t.Fatalf("expr column name = %q, want total", got)
	}
	if got := sh.Cell(0, 2); got != "3" {
		t.Fatalf("first expr value = %q, want 3", got)
	}
	if got := sh.Cell(1, 2); got != "8" {
		t.Fatalf("second expr value = %q, want 8", got)
	}
	if got := sh.Columns[2].Kind; got != sheet.KindInt {
		t.Fatalf("expr column kind = %s, want int", got)
	}
	if got := sh.CursorCol; got != 2 {
		t.Fatalf("cursor col = %d, want 2", got)
	}
}

func TestHandleKeyAddsSparklineColumn(t *testing.T) {
	sh := sheet.New("metrics.csv", "/tmp/metrics.csv", []string{"city", "a", "b", "c"})
	sh.AddRow([]string{"Tokyo", "1", "5", "9"})
	sh.AddRow([]string{"Osaka", "3", "3", "3"})
	sh.InferColumnKinds()

	app := New(sh, nil)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '*', tcell.ModNone))

	if got := len(sh.Columns); got != 5 {
		t.Fatalf("len(columns) = %d, want 5", got)
	}
	if got := sh.Columns[4].Name; got != "sparkline" {
		t.Fatalf("sparkline column name = %q, want sparkline", got)
	}
	if got := sh.Cell(0, 4); got != "▁▅█" {
		t.Fatalf("sparkline value = %q, want ▁▅█", got)
	}
	if got := sh.Cell(1, 4); got != "▅▅▅" {
		t.Fatalf("flat sparkline value = %q, want ▅▅▅", got)
	}
}

func TestHandleKeyEditsCellAndUndoRestoresValue(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.AddRow([]string{"Bob", "20"})
	sh.InferColumnKinds()

	app := New(sh, nil)
	sh.SetCursorRow(1)
	sh.SetCursorCol(1)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'e', tcell.ModNone))
	for i := 0; i < 2; i++ {
		app.HandleKey(tcell.NewEventKey(tcell.KeyBackspace2, 0, tcell.ModNone))
	}
	for _, r := range "25" {
		app.HandleKey(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if got := sh.Cell(1, 1); got != "25" {
		t.Fatalf("edited cell value = %q, want 25", got)
	}
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'U', tcell.ModNone))
	if got := sh.Cell(1, 1); got != "20" {
		t.Fatalf("cell value after undo = %q, want 20", got)
	}
}

func TestHandleKeyUndoRestoresDeletedRows(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.AddRow([]string{"Bob", "20"})
	sh.AddRow([]string{"Carol", "10"})

	app := New(sh, nil)
	sh.SetCursorRow(1)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'd', tcell.ModNone))
	if got := len(sh.Rows); got != 2 {
		t.Fatalf("len(rows) after delete = %d, want 2", got)
	}
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'U', tcell.ModNone))
	if got := len(sh.Rows); got != 3 {
		t.Fatalf("len(rows) after undo = %d, want 3", got)
	}
	if got := sh.Cell(1, 0); got != "Bob" {
		t.Fatalf("row restored after undo = %q, want Bob", got)
	}
	if !strings.Contains(sh.Status, "undid delete rows") {
		t.Fatalf("status after undo = %q, want undo status", sh.Status)
	}
}

func TestHandleKeyRedoReappliesDeletedRows(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.AddRow([]string{"Bob", "20"})
	sh.AddRow([]string{"Carol", "10"})

	app := New(sh, nil)
	sh.SetCursorRow(1)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'd', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'U', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'R', tcell.ModNone))
	if got := len(sh.Rows); got != 2 {
		t.Fatalf("len(rows) after redo = %d, want 2", got)
	}
	if got := sh.Cell(1, 0); got != "Carol" {
		t.Fatalf("row after redo = %q, want Carol", got)
	}
	if !strings.Contains(sh.Status, "redid delete rows") {
		t.Fatalf("status after redo = %q, want redo status", sh.Status)
	}
}

func TestHandleKeyOpensCommandLogSheet(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.InferColumnKinds()

	app := New(sh, nil)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '=', tcell.ModNone))
	for _, r := range "double := age * 2" {
		app.HandleKey(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'P', tcell.ModNone))
	if got := app.Sheet.Name; got != "cmdlog" {
		t.Fatalf("cmdlog sheet name = %q, want cmdlog", got)
	}
	if got := app.Sheet.Cell(0, 2); got != "expr-col" {
		t.Fatalf("first logged command = %q, want expr-col", got)
	}
	if got := app.Sheet.Cell(1, 2); got != "cmdlog" {
		t.Fatalf("second logged command = %q, want cmdlog", got)
	}
}

func TestCommandPaletteExecutesSelectedCommand(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"city", "age"})
	sh.AddRow([]string{"Tokyo", "30"})
	sh.AddRow([]string{"Tokyo", "20"})
	sh.AddRow([]string{"Osaka", "10"})
	sh.InferColumnKinds()

	app := New(sh, nil)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, ':', tcell.ModNone))
	for _, r := range "freq" {
		app.HandleKey(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))

	if got := app.Sheet.Name; got != "freq:people.csv:city" {
		t.Fatalf("sheet name after palette frequency = %q, want freq:people.csv:city", got)
	}
	if got := app.Sheet.MetaKind; got != "freq" {
		t.Fatalf("meta kind = %q, want freq", got)
	}
}

func TestCommandPaletteCanLaunchInputCommand(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.InferColumnKinds()

	app := New(sh, nil)
	app.HandleKey(tcell.NewEventKey(tcell.KeyCtrlP, 0, tcell.ModNone))
	for _, r := range "rename" {
		app.HandleKey(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	for _, r := range "person" {
		app.HandleKey(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))

	if got := sh.Columns[0].Name; got != "person" {
		t.Fatalf("renamed column = %q, want person", got)
	}
}

func TestDrawRendersCommandPaletteOverlay(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.InferColumnKinds()

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(80, 12)

	app := New(sh, screen)
	app.beginCommandPalette()
	for _, r := range "freq" {
		app.handleCommandPaletteKey(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	app.Draw()

	lines := snapshot(screen, 80, 12)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		"Command Palette: freq",
		"frequency",
		"[F]",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("screen missing %q:\n%s", want, joined)
		}
	}
}

func TestMenuExecutesSelectedCommand(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.AddRow([]string{"Bob", "20"})
	sh.InferColumnKinds()

	app := New(sh, nil)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, ';', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
	for i := 0; i < 7; i++ {
		app.HandleKey(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
	}
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if got := sh.Cell(0, 0); got != "Bob" {
		t.Fatalf("first row after menu sort desc = %q, want Bob", got)
	}
}

func TestDrawRendersMenuOverlay(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.InferColumnKinds()

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(80, 12)

	app := New(sh, screen)
	app.beginMenu()
	app.Draw()

	lines := snapshot(screen, 80, 12)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		" File ",
		" Edit ",
		"> Save",
		"Quit",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("screen missing %q:\n%s", want, joined)
		}
	}
}

func TestHandleMouseClickMovesCursor(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.AddRow([]string{"Bob", "20"})
	sh.InferColumnKinds()

	app := New(sh, nil)
	secondColX := rowPrefixWidth + app.colWidths[0] + 3 + 1
	app.HandleMouse(tcell.NewEventMouse(secondColX, 4, tcell.Button1, 0))
	app.HandleMouse(tcell.NewEventMouse(secondColX, 4, tcell.ButtonNone, 0))

	if sh.CursorRow != 1 || sh.CursorCol != 1 {
		t.Fatalf("cursor = (%d,%d), want (1,1)", sh.CursorRow, sh.CursorCol)
	}
}

func TestHandleMouseWheelScrollsRows(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name"})
	for _, name := range []string{"Alice", "Bob", "Carol", "Dave", "Eve", "Frank"} {
		sh.AddRow([]string{name})
	}

	app := New(sh, nil)
	app.HandleMouse(tcell.NewEventMouse(1, 4, tcell.WheelDown, 0))
	if sh.CursorRow == 0 {
		t.Fatalf("expected wheel down to move cursor, got row %d", sh.CursorRow)
	}
}

func TestHandleMouseDragSelectsRowRange(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name"})
	for _, name := range []string{"Alice", "Bob", "Carol", "Dave"} {
		sh.AddRow([]string{name})
	}

	app := New(sh, nil)
	x := rowPrefixWidth + 1
	app.HandleMouse(tcell.NewEventMouse(x, 3, tcell.Button1, 0))
	app.HandleMouse(tcell.NewEventMouse(x, 5, tcell.Button1, 0))
	app.HandleMouse(tcell.NewEventMouse(x, 5, tcell.ButtonNone, 0))

	if got := sh.SelectedCount(); got != 3 {
		t.Fatalf("SelectedCount = %d, want 3", got)
	}
	for _, row := range []int{0, 1, 2} {
		if !sh.IsSelected(row) {
			t.Fatalf("expected row %d to be selected", row)
		}
	}
}

func TestMacroReplayReappliesRecordedCommand(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.AddRow([]string{"Bob", "20"})
	sh.InferColumnKinds()
	sh.SetCursorCol(1)

	app := New(sh, nil)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'z', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'z', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '[', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'z', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'z', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'U', tcell.ModNone))
	if got := sh.Cell(0, 0); got != "Alice" {
		t.Fatalf("first row after undo = %q, want Alice", got)
	}

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'z', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'Z', tcell.ModNone))
	if got := sh.Cell(0, 0); got != "Bob" {
		t.Fatalf("first row after macro replay = %q, want Bob", got)
	}
}

func TestMacroReplayPreservesInputCommands(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "city"})
	sh.AddRow([]string{"Alice", "Tokyo"})
	sh.AddRow([]string{"Bob", "Osaka"})
	sh.InferColumnKinds()

	app := New(sh, nil)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'z', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'z', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '/', tcell.ModNone))
	for _, r := range "osaka" {
		app.HandleKey(tcell.NewEventKey(tcell.KeyRune, r, tcell.ModNone))
	}
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'z', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'z', tcell.ModNone))

	sh.ClearSearch()
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'z', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'Z', tcell.ModNone))
	if got := sh.SearchState.Query; got != "osaka" {
		t.Fatalf("search query after macro replay = %q, want osaka", got)
	}
	if len(sh.SearchState.Matches) == 0 {
		t.Fatal("expected replayed search to restore matches")
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
		"Enter open-row",
		"sort age↑",
		"search \"o\"",
		"sel 1",
		"= expr-col",
		"cell \"20\"",
		"& join",
		"f freeze",
		"m melt",
		"D dedupe",
		"W pivot",
		"P cmdlog",
		"R redo",
		"U undo",
		"y / C copy",
		"e edit-cell",
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
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, '&', tcell.ModNone))
	if got := app.Sheet.Name; got != "join:freeze:people.csv+people.csv:city=city" {
		t.Fatalf("join sheet name = %q, want join:freeze:people.csv+people.csv:city=city", got)
	}
	if quit := app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone)); quit {
		t.Fatal("q should pop join sheet before quitting")
	}
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
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
	if got := app.Sheet.Name; got != "filter:people.csv:city=Tokyo" {
		t.Fatalf("filter sheet name = %q, want filter:people.csv:city=Tokyo", got)
	}
	if got := len(app.Sheet.Rows); got != 2 {
		t.Fatalf("filtered row count = %d, want 2", got)
	}
	if got := app.Sheet.Cell(1, 1); got != "20" {
		t.Fatalf("second filtered age = %q, want 20", got)
	}
	if quit := app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone)); quit {
		t.Fatal("q should pop filtered sheet before quitting")
	}
	if got := app.Sheet.Name; got != "freq:people.csv:city" {
		t.Fatalf("sheet after filtered pop = %q, want freq:people.csv:city", got)
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

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'm', tcell.ModNone))
	if got := app.Sheet.Name; got != "melt:people.csv:city" {
		t.Fatalf("melt sheet name = %q, want melt:people.csv:city", got)
	}
	_ = app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'q', tcell.ModNone))

	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'W', tcell.ModNone))
	if got := app.Sheet.Name; got != "pivot:people.csv:city" {
		t.Fatalf("pivot sheet name = %q, want pivot:people.csv:city", got)
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

func TestHandleKeySidebarSwitchesToEarlierSheet(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"city", "age"})
	sh.AddRow([]string{"Tokyo", "30"})
	sh.AddRow([]string{"Tokyo", "20"})
	sh.AddRow([]string{"Osaka", "10"})
	sh.InferColumnKinds()

	app := New(sh, nil)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'F', tcell.ModNone))
	if got := app.Sheet.Name; got != "freq:people.csv:city" {
		t.Fatalf("sheet name after frequency = %q, want freq:people.csv:city", got)
	}

	app.HandleKey(tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))

	if app.sidebarVisible {
		t.Fatal("expected sidebar to close after switching sheets")
	}
	if got := app.Sheet.Name; got != "people.csv" {
		t.Fatalf("sheet after sidebar switch = %q, want people.csv", got)
	}
	if len(app.stack) != 1 {
		t.Fatalf("stack depth after sidebar switch = %d, want 1", len(app.stack))
	}
}

func TestDrawRendersSidebar(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"city", "age"})
	sh.AddRow([]string{"Tokyo", "30"})
	sh.AddRow([]string{"Tokyo", "20"})
	sh.AddRow([]string{"Osaka", "10"})
	sh.InferColumnKinds()

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(100, 12)

	app := New(sh, screen)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'F', tcell.ModNone))
	app.HandleKey(tcell.NewEventKey(tcell.KeyTab, 0, tcell.ModNone))
	app.Draw()

	lines := snapshot(screen, 100, 12)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		" Sheets ",
		"people.csv",
		"freq:people.csv:city",
		"sidebar",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("screen missing %q:\n%s", want, joined)
		}
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
		"Enter open-row",
		"= expr-col",
		"& join",
		"f freeze",
		"m melt",
		"D dedupe",
		"d delete",
		"P cmdlog",
		"R redo",
		"U undo",
		"e edit-cell",
		"F freq",
		"I describe",
		"W pivot",
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
	screen.SetSize(560, 8)

	app := New(sh, screen)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'M', tcell.ModNone))
	app.Draw()

	lines := snapshot(screen, 560, 8)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		"columns:people.csv",
		"stack 2",
		"Enter open-row",
		"= expr-col",
		"& join",
		"f freeze",
		"m melt",
		"W pivot",
		"V sheets",
		"P cmdlog",
		"R redo",
		"U undo",
		"? commands",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("screen missing %q:\n%s", want, joined)
		}
	}
}

func TestDrawRendersExpressionInputPrompt(t *testing.T) {
	sh := sheet.New("orders.csv", "/tmp/orders.csv", []string{"qty", "price"})
	sh.AddRow([]string{"2", "1.5"})

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(200, 8)

	app := New(sh, screen)
	app.beginInput(inputModeExpr, "total := qty * price")
	app.Draw()

	lines := snapshot(screen, 200, 8)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		"Expr column: total := qty * price",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("screen missing %q:\n%s", want, joined)
		}
	}
}

func TestDrawRendersEditCellInputPrompt(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(200, 8)

	app := New(sh, screen)
	app.beginInput(inputModeEditCell, "30")
	app.Draw()

	lines := snapshot(screen, 200, 8)
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "Edit cell: 30") {
		t.Fatalf("screen missing edit prompt:\n%s", joined)
	}
}

func TestHandleKeyOpensGraphSheet(t *testing.T) {
	sh := sheet.New("sales.csv", "/tmp/sales.csv", []string{"dept", "amount"})
	sh.AddRow([]string{"A", "10"})
	sh.AddRow([]string{"B", "20"})
	sh.AddRow([]string{"C", "15"})
	sh.InferColumnKinds()

	app := New(sh, nil)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'v', tcell.ModNone))

	if got := app.Sheet.MetaKind; got != "graph" {
		t.Fatalf("meta kind = %q, want graph", got)
	}
	if app.Sheet.Graph == nil {
		t.Fatal("expected graph metadata")
	}
	if got := app.Sheet.Graph.Kind; got != sheet.GraphBar {
		t.Fatalf("graph kind = %s, want bar", got)
	}
	if got := app.Sheet.Name; got != "graph:sales.csv:amount" {
		t.Fatalf("graph sheet name = %q, want graph:sales.csv:amount", got)
	}
}

func TestDrawRendersGraphCanvas(t *testing.T) {
	sh := sheet.New("sales.csv", "/tmp/sales.csv", []string{"dept", "amount"})
	sh.AddRow([]string{"A", "10"})
	sh.AddRow([]string{"B", "20"})
	sh.AddRow([]string{"C", "15"})
	sh.InferColumnKinds()

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(200, 12)

	app := New(sh, screen)
	app.HandleKey(tcell.NewEventKey(tcell.KeyRune, 'v', tcell.ModNone))
	app.Draw()

	lines := snapshot(screen, 200, 12)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		"graph:sales.csv:amount: 3 point(s) [bar]",
		"amount by dept [bar]",
		"graph bar",
		"x dept",
		"y amount",
		"point 1/3",
		"dept [A .. C]",
		"@",
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

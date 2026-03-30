package tui

import (
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
}

func TestDrawRendersSortSearchAndSelectionStatus(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.AddRow([]string{"Bob", "20"})
	sh.InferColumnKinds()
	sh.ToggleSelected(1)
	sh.ToggleSort(1, sheet.SortAsc)
	sh.Search("o")

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(80, 8)

	app := New(sh, screen)
	app.Draw()

	lines := snapshot(screen, 80, 8)
	joined := strings.Join(lines, "\n")
	for _, want := range []string{
		"* ",
		"sort age↑",
		"search \"o\"",
		"sel 1",
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

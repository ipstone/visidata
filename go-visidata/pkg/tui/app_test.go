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

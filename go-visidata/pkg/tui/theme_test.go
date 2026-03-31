package tui

import (
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/ipstone/visidata/go-visidata/pkg/sheet"
)

func TestResolveThemeFallsBackToDefault(t *testing.T) {
	if got := ResolveTheme("amber").Name; got != "amber" {
		t.Fatalf("ResolveTheme(amber) = %q, want amber", got)
	}
	if got := ResolveTheme("missing").Name; got != "default" {
		t.Fatalf("ResolveTheme(missing) = %q, want default", got)
	}
}

func TestDrawUsesSelectedThemeStyles(t *testing.T) {
	sh := sheet.New("people.csv", "/tmp/people.csv", []string{"name", "age"})
	sh.AddRow([]string{"Alice", "30"})
	sh.InferColumnKinds()

	screen := tcell.NewSimulationScreen("UTF-8")
	if err := screen.Init(); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	defer screen.Fini()
	screen.SetSize(80, 8)

	app := NewWithOptions(sh, screen, Options{Theme: "amber"})
	app.Draw()

	_, _, style, _ := screen.GetContent(0, 0)
	fg, bg, _ := style.Decompose()
	if fg != builtinThemes["amber"].HeaderFG {
		t.Fatalf("header fg = %v, want %v", fg, builtinThemes["amber"].HeaderFG)
	}
	if bg != builtinThemes["amber"].HeaderBG {
		t.Fatalf("header bg = %v, want %v", bg, builtinThemes["amber"].HeaderBG)
	}
}

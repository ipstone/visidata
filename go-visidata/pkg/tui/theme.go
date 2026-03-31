package tui

import "github.com/gdamore/tcell/v2"

type Options struct {
	Theme string
}

type Theme struct {
	Name      string
	HeaderFG  tcell.Color
	HeaderBG  tcell.Color
	StatusFG  tcell.Color
	StatusBG  tcell.Color
	BodyFG    tcell.Color
	BodyBG    tcell.Color
	AccentFG  tcell.Color
	SearchBG  tcell.Color
	CursorFG  tcell.Color
	CursorBG  tcell.Color
	OverlayFG tcell.Color
	OverlayBG tcell.Color
}

var builtinThemes = map[string]Theme{
	"default": {
		Name:      "default",
		HeaderFG:  tcell.ColorWhite,
		HeaderBG:  tcell.ColorBlack,
		StatusFG:  tcell.ColorBlack,
		StatusBG:  tcell.ColorWhite,
		BodyFG:    tcell.ColorWhite,
		BodyBG:    tcell.ColorBlack,
		AccentFG:  tcell.ColorLightGreen,
		SearchBG:  tcell.ColorDarkCyan,
		CursorFG:  tcell.ColorBlack,
		CursorBG:  tcell.ColorWhite,
		OverlayFG: tcell.ColorWhite,
		OverlayBG: tcell.ColorDarkSlateGray,
	},
	"amber": {
		Name:      "amber",
		HeaderFG:  tcell.ColorBlack,
		HeaderBG:  tcell.ColorYellow,
		StatusFG:  tcell.ColorBlack,
		StatusBG:  tcell.ColorLightGoldenrodYellow,
		BodyFG:    tcell.ColorWheat,
		BodyBG:    tcell.ColorBlack,
		AccentFG:  tcell.ColorOrange,
		SearchBG:  tcell.ColorSaddleBrown,
		CursorFG:  tcell.ColorBlack,
		CursorBG:  tcell.ColorOrange,
		OverlayFG: tcell.ColorBlack,
		OverlayBG: tcell.ColorKhaki,
	},
	"ocean": {
		Name:      "ocean",
		HeaderFG:  tcell.ColorWhite,
		HeaderBG:  tcell.ColorTeal,
		StatusFG:  tcell.ColorWhite,
		StatusBG:  tcell.ColorNavy,
		BodyFG:    tcell.ColorLightCyan,
		BodyBG:    tcell.ColorBlack,
		AccentFG:  tcell.ColorAqua,
		SearchBG:  tcell.ColorDarkBlue,
		CursorFG:  tcell.ColorWhite,
		CursorBG:  tcell.ColorTeal,
		OverlayFG: tcell.ColorWhite,
		OverlayBG: tcell.ColorDarkCyan,
	},
}

func ResolveTheme(name string) Theme {
	if theme, ok := builtinThemes[name]; ok {
		return theme
	}
	return builtinThemes["default"]
}

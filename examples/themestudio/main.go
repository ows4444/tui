// Command themestudio is a workshop for themes: pick one of the twenty built-in
// themes, change any of its fifteen colours with red, green and blue sliders or
// a hex code, and watch a sample screen redraw at once. Contrast is checked as
// you go, and the result is saved as a theme file that theme.Parse (and so
// any program that loads a theme file) reads back.
//
// The preview is drawn from the theme the file would give, not from a copy, so
// what you see is what the file holds. Nothing is written unless you save.
//
// Mouse: click a colour in the list, click or drag along a slider, wheel over
// one to nudge it, click the hex code to type one, click the arrows by the
// theme name to change base, click Border or Glyphs to cycle them, click Save.
// Keyboard: Up and Down pick a colour, Tab and Shift+Tab pick a slider or the
// hex code, Left and Right change a value (PgUp and PgDn by sixteen), Enter
// types a hex code, [ and ] change the base theme, x resets the colour, 0
// resets everything, b and g cycle the border and glyphs, s saves, ? shows
// help, q or Ctrl+C quits.
//
//	go run ./examples/themestudio
//	go run ./examples/themestudio -out mytheme.json
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ows4444/tui"
)

func main() {
	out := flag.String("out", "theme.json", "where s saves the theme file")
	flag.Parse()
	m := newModel()
	m.path = *out
	if _, err := tui.NewProgram(m, tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

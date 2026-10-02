package theme_test

import (
	"fmt"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/theme"
)

type app struct{ theme theme.Theme }

func (a app) Init() tui.Cmd { return nil }

func (a app) View() string { return "" }

// Update switches theme when the terminal answers the background query that
// tui.WithBackgroundDetection sent at startup.
func (a app) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if th, ok := theme.Detect(msg, theme.DarkTheme()); ok {
		a.theme = th
	}
	return a, nil
}

// The recipe: start with a default theme, ask for the background with
// tui.WithBackgroundDetection, and let theme.Detect pick Light or Dark when
// the answer (or the timeout) arrives. Any later change is one assignment.
//
//	tui.NewProgram(app{theme: theme.DarkTheme()}, tui.WithBackgroundDetection(200*time.Millisecond))
func ExampleDetect_switchTheme() {
	var m tui.Model = app{theme: theme.DarkTheme()}

	// A light terminal answers with a near-white background.
	m, _ = m.Update(tui.BackgroundColorEvent{R: 250, G: 250, B: 250})
	fmt.Println("light background -> Text is Light's:", m.(app).theme.Text == theme.LightTheme().Text)

	// A dark one answers dark.
	m, _ = m.Update(tui.BackgroundColorEvent{R: 10, G: 10, B: 10})
	fmt.Println("dark background  -> Text is Dark's:", m.(app).theme.Text == theme.DarkTheme().Text)

	// Silence keeps the fallback and tells the app to stop waiting.
	m, _ = m.Update(tui.BackgroundUnknownMsg{})
	fmt.Println("no answer        -> Text is Dark's:", m.(app).theme.Text == theme.DarkTheme().Text)

	// Unrelated messages change nothing.
	m, _ = m.Update(tui.Key{Type: tui.KeyEnter})
	fmt.Println("other message    -> unchanged:", m.(app).theme.Text == theme.DarkTheme().Text)
	// Output:
	// light background -> Text is Light's: true
	// dark background  -> Text is Dark's: true
	// no answer        -> Text is Dark's: true
	// other message    -> unchanged: true
}

// Package uitest holds the checks the example programs share, so that each one
// is a single line in the example's own tests.
package uitest

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/tuitest"
)

// Check builds the model and tests that it follows the theme and speaks to a
// screen reader: it takes the Program's theme through SetTheme, looks the same in
// words in the light theme as in the dark one but is drawn in other colours, and
// its Linearize text is plain.
func Check(t *testing.T, build func() tui.Model) {
	t.Helper()
	dark := sized(t, build())
	th, ok := build().(tui.Themeable)
	if !ok {
		t.Fatal("the model does not implement tui.Themeable (SetTheme)")
	}
	light := sized(t, th.SetTheme(theme.LightTheme()))
	// a theme may draw its borders differently, so compare the words alone
	if d, l := ui.Linearize(dark.View()), ui.Linearize(light.View()); d != l {
		t.Errorf("the light theme changes the words on screen:\n--- dark\n%s\n--- light\n%s", d, l)
	}
	if dark.View() == light.View() {
		t.Error("the light theme draws exactly like the dark one: the model is not using its theme")
	}
	l, ok := dark.(tui.Linearizer)
	if !ok {
		t.Fatal("the model does not implement tui.Linearizer")
	}
	text := l.Linearize()
	switch {
	case strings.TrimSpace(text) == "":
		t.Error("Linearize is empty")
	case strings.ContainsRune(text, 0x1b):
		t.Error("Linearize holds an escape sequence")
	}
	for _, line := range strings.Split(text, "\n") {
		if line != strings.TrimSpace(line) || line == "" {
			t.Errorf("Linearize line %q is padded or empty", line)
		}
	}
	accessible(t, build)
}

// accessible runs the model as a Program in accessible mode, as ACCESSIBLE=1
// does, and checks that the transcript has words in it.
func accessible(t *testing.T, build func() tui.Model) {
	t.Helper()
	s := tuitest.New(build(), 80, 24, tui.WithAccessible(true))
	defer s.Close()
	first := strings.Fields(strings.Join(s.Screen(), " "))
	if len(first) == 0 {
		t.Error("accessible mode wrote nothing")
	}
}

// sized gives the model an 80x24 terminal.
func sized(t *testing.T, m tui.Model) tui.Model {
	t.Helper()
	m, _ = m.Update(tui.ResizeMsg{Width: 80, Height: 24})
	return m
}

// CheckHelp tests a help screen: open is the key that opens it, want is a text
// that only the help screen shows. Any key closes it, and Ctrl+C still quits.
func CheckHelp(t *testing.T, build func() tui.Model, open tui.Key, want string) {
	t.Helper()
	m := sized(t, build())
	if strings.Contains(ansi.StripANSI(m.View()), want) {
		t.Fatalf("%q is on the first screen already", want)
	}
	opened, _ := m.Update(open)
	if !strings.Contains(ansi.StripANSI(opened.View()), want) {
		t.Fatalf("the help did not open:\n%s", ansi.StripANSI(opened.View()))
	}
	if _, cmd := opened.Update(tui.Key{Type: tui.KeyCtrlC}); cmd == nil {
		t.Error("Ctrl+C does not quit from the help")
	} else if _, ok := cmd().(tui.QuitMsg); !ok {
		t.Error("Ctrl+C from the help did not quit")
	}
	closed, _ := opened.Update(tui.Key{Type: tui.KeyRunes, Text: "x"})
	if strings.Contains(ansi.StripANSI(closed.View()), want) {
		t.Errorf("a key did not close the help:\n%s", ansi.StripANSI(closed.View()))
	}
}

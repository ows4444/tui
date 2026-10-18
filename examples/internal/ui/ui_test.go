package ui

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/tuitest"
)

// probe remembers the theme the Program gave it and shows its name.
type probe struct{ th Theming }

func (p probe) Init() tui.Cmd                       { return nil }
func (p probe) Update(tui.Msg) (tui.Model, tui.Cmd) { return p, nil }
func (p probe) SetTheme(t theme.Theme) tui.Model    { p.th = p.th.Set(t); return p }
func (p probe) View() string {
	if p.th.Get().Primary == theme.NordTheme().Primary {
		return "nord"
	}
	return "other"
}

func TestTheProgramHandsTheThemeToTheModelBeforeTheFirstFrame(t *testing.T) {
	s := tuitest.New(probe{}, 20, 3, tui.WithTheme(theme.Pair(theme.NordTheme())))
	defer s.Close()
	if !s.WaitForText("nord", 3e9) {
		t.Fatalf("the model never got the theme:\n%s", strings.Join(s.Screen(), "\n"))
	}
}

func TestThemingDefaultsToDark(t *testing.T) {
	var th Theming
	if th.Get().Primary != theme.DarkTheme().Primary {
		t.Fatal("the zero Theming is not the dark theme")
	}
	if th.Set(theme.LightTheme()).Get().Primary != theme.LightTheme().Primary {
		t.Fatal("Set did not hold the theme")
	}
}

func TestLinearizeKeepsWordsAndDropsDecoration(t *testing.T) {
	view := "\x1b[1m╭──────────╮\x1b[0m\n│ Hello    │\n│ ░░░░     │\n╰──────────╯\n\n  a · b   \n"
	if got, want := Linearize(view), "Hello\na · b"; got != want {
		t.Fatalf("%q, want %q", got, want)
	}
}

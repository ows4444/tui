package appshell

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/theme"
)

// themedRoot is a root model built on appshell. Its SetTheme is all the app
// code there is: one call to the shell's own SetTheme.
type themedRoot struct{ shell Model }

func (r themedRoot) Init() tui.Cmd { return nil }
func (r themedRoot) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	s, c := r.shell.Update(msg)
	r.shell = s
	return r, c
}
func (r themedRoot) View() string { return r.shell.View() }
func (r themedRoot) SetTheme(t theme.Theme) tui.Model {
	r.shell = r.shell.SetTheme(t)
	return r
}

var _ tui.Themeable = themedRoot{}

// Criterion #1 (spec #32): a light-background answer reaches the shell and
// its children as the Light theme, through Themeable.SetTheme.
func TestSetThemeRendersShellAndChildrenInLight(t *testing.T) {
	var root tui.Model = themedRoot{shell: New("Title", 20, 3)}
	dark := root.View()

	root = root.(tui.Themeable).SetTheme(theme.LightTheme())
	got := root.(themedRoot).shell
	if got.Theme != theme.LightTheme() {
		t.Fatalf("shell theme not Light")
	}
	want := New("Title", 20, 3)
	want.Theme = theme.LightTheme()
	want.Input.PlaceholderStyle = got.Input.PlaceholderStyle
	want.Input.TextStyle = got.Input.TextStyle
	if root.View() != want.View() {
		t.Error("shell did not render with the Light theme")
	}
	if got.Input.TextStyle == New("x", 1, 1).Input.TextStyle {
		t.Error("Input child was not re-themed")
	}
	_ = dark
}

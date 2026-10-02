package textarea

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func TestSetThemeStylesPlaceholderAndKeepsContent(t *testing.T) {
	th := theme.DraculaTheme()
	m := New()
	m.Placeholder = "name"
	m = m.SetTheme(th)
	want := ansi.NewStyle().Faint().Foreground(th.Muted).Render("name")
	if got := m.View(); !strings.Contains(got, want) {
		t.Errorf("View = %q, want it to contain the themed placeholder %q", got, want)
	}
	if m.CursorStyle.Render("x") != ansi.NewStyle().Reverse().Render("x") {
		t.Error("cursor lost reverse video")
	}
}

func TestSetThemeLeavesTheOriginalAlone(t *testing.T) {
	m := New()
	before := m.PlaceholderStyle.Render("p")
	_ = m.SetTheme(theme.DraculaTheme())
	if m.PlaceholderStyle.Render("p") != before {
		t.Error("SetTheme changed the receiver")
	}
}

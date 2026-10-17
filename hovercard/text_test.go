package hovercard

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
)

func TestContentTextIsShownInsteadOfContentWithItsStyles(t *testing.T) {
	m := New("Title", "ignored", hittest.Rect{X: 2, Y: 1, W: 4, H: 1})
	m.ContentText = ansi.NewText("see ", ansi.Style{}).Append("docs", ansi.NewStyle().Underline())
	m.Show()
	out := m.Render(strings.Repeat(strings.Repeat(".", 30)+"\n", 8))
	if strings.Contains(out, "ignored") {
		t.Error("Content was drawn although ContentText is set")
	}
	if !strings.Contains(out, "\x1b[4mdocs") {
		t.Errorf("the underline was lost: %q", out)
	}
	if got := m.Linearize(); !strings.Contains(got, "see docs") {
		t.Errorf("Linearize = %q", got)
	}
}

func TestContentTextWrapsAtWidth(t *testing.T) {
	m := New("", "", hittest.Rect{X: 0, Y: 0, W: 2, H: 1})
	m.Width = 8
	m.ContentText = ansi.NewText("alpha beta gamma", ansi.Style{})
	m.Show()
	out := ansi.StripANSI(m.Render(strings.Repeat(strings.Repeat(".", 30)+"\n", 8)))
	if !strings.Contains(out, "alpha") || !strings.Contains(out, "gamma") || strings.Contains(out, "alpha beta") {
		t.Errorf("not wrapped at 8:\n%s", out)
	}
}

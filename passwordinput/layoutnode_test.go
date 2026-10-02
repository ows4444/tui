package passwordinput

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func TestLayoutNodeNeverDrawsThePassword(t *testing.T) {
	m := New()
	m.Prompt = "Password: "
	m.SetValue("hunter2-secret")
	m.Focus()
	for _, s := range []layout.Size{{W: 40, H: 1}, {W: 14, H: 1}, {W: 8, H: 2}} {
		out := ansi.StripANSI(m.LayoutNode().Render(s))
		if strings.Contains(out, "hunter") || strings.Contains(out, "secret") {
			t.Errorf("%v drew the password: %q", s, out)
		}
		for _, l := range strings.Split(out, "\n") {
			if len([]rune(l)) != s.W {
				t.Errorf("%v: row width %d", s, len([]rune(l)))
			}
		}
	}
	if !strings.Contains(ansi.StripANSI(m.LayoutNode().Render(layout.Size{W: 40, H: 1})), "•") {
		t.Error("expected mask characters")
	}
	// The promoted textinput node would draw it; pin the premise.
	if !strings.Contains(ansi.StripANSI(m.Model.LayoutNode().Render(layout.Size{W: 40, H: 1})), "hunter2-secret") {
		t.Fatal("test premise changed: the embedded LayoutNode no longer draws the value")
	}
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{W: 10 + 14 + 1, H: 1}) {
		t.Errorf("Measure = %v", got)
	}
}

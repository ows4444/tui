package wizard

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

func exact(t *testing.T, out string, wd, ht int) []string {
	t.Helper()
	lines := strings.Split(out, "\n")
	if len(lines) != ht {
		t.Fatalf("%dx%d: %d rows:\n%s", wd, ht, len(lines), out)
	}
	for i, l := range lines {
		if ansi.Width(l) != wd {
			t.Fatalf("%dx%d: row %d is %d wide: %q", wd, ht, i, ansi.Width(l), l)
		}
	}
	return lines
}

var sizes = []layout.Size{{W: 30, H: 8}, {W: 12, H: 3}, {W: 5, H: 1}, {W: 40, H: 20}, {W: 1, H: 1}}

func allSizes(t *testing.T, n layout.Node) {
	t.Helper()
	for _, s := range sizes {
		exact(t, n.Render(s), s.W, s.H)
	}
	exact(t, n.Render(layout.Size{W: 9, H: 2}), 9, 2)
}

func TestLayoutNodeKeepsTheCurrentStepVisible(t *testing.T) {
	m := New("Account", "Profile", "Preferences", "Billing", "Review", "Confirm")
	for i := 0; i < 4; i++ {
		if err := m.Next(nil); err != nil {
			t.Fatal(err)
		}
	}
	out := ansi.StripANSI(m.LayoutNode(theme.DarkTheme()).Render(layout.Size{W: 26, H: 1}))
	exact(t, out, 26, 1)
	if !strings.Contains(out, "Review") {
		t.Errorf("current step not visible: %q", out)
	}
	wide := ansi.StripANSI(m.LayoutNode(theme.DarkTheme()).Render(layout.Size{W: 120, H: 1}))
	if !strings.Contains(wide, "Account") || !strings.Contains(wide, "Confirm") {
		t.Errorf("a wide slot should show every step: %q", wide)
	}
	if m.Current() != 4 {
		t.Error("rendering changed the Model")
	}
	allSizes(t, m.LayoutNode(theme.DarkTheme()))
	if got := New().LayoutNode(theme.DarkTheme()).Measure(layout.Unconstrained()); got != (layout.Size{}) {
		t.Errorf("empty Measure = %v", got)
	}
}
